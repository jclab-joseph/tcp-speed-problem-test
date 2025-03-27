package main

import (
	crand "crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"embed"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"github.com/jclab-joseph/tcp-speed-problem-test/internal/certutil"
	"github.com/jclab-joseph/tcp-speed-problem-test/pkg/tcpinfo"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/quic-go/qlog"
	"io/fs"
	"log"
	"log/slog"
	randv2 "math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

//go:embed frontend
var frontendFiles embed.FS

type tcpInfoCollector struct {
	handler http.Handler
}

func (t *tcpInfoCollector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Store original connection hijacker
	hj, ok := w.(http.Hijacker)
	if !ok {
		log.Printf("Hijacking not supported")
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	conn, bufrw, err := hj.Hijack()
	if err != nil {
		log.Printf("Hijacking not supported: %+v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create new response writer with wrapped connection
	newWriter := &responseWriter{
		bufrw:  bufrw,
		header: make(http.Header),
	}

	// Serve the original handler
	reqCtx, appCtx := WithTcpCtx(r.Context())
	appCtx.NativeConn = conn
	t.handler.ServeHTTP(newWriter, r.WithContext(reqCtx))
	newWriter.Flush()
	_ = conn.Close()
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	tcpCtx := GetTcpCtx(r.Context())

	var size int = 16
	sizeStr := r.URL.Query().Get("size")
	if sizeStr != "" {
		n, err := strconv.ParseInt(sizeStr, 10, 32)
		if err != nil {
			log.Printf("parse int failed: value=[%s]: %+v", sizeStr, err)
		} else {
			size = int(n)
		}
	}

	var seed [32]byte
	_, _ = crand.Read(seed[:])
	rnd := randv2.NewChaCha8(seed)

	footerSize := 4096
	dummySize := size * 1024 * 1024
	totalBytes := dummySize + footerSize

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.Itoa(totalBytes))
	if tcpCtx != nil {
		w.Header().Set("Connection", "close")
	}
	w.WriteHeader(200)

	chunk := make([]byte, 128*1024)
	for i := 0; i < dummySize/len(chunk); i++ {
		_, _ = rnd.Read(chunk)
		for i := range chunk {
			if chunk[i] == 0 {
				chunk[i] = 1
			}
		}

		if _, err := w.Write(chunk); err != nil {
			log.Printf("write failed 1: %+v", err)
			break
		}

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}

	footerBuffer := make([]byte, footerSize)

	if tcpCtx != nil {
		tcpInfo, err := tcpinfo.GetTcpInfo(tcpCtx.NativeConn)
		if err != nil {
			log.Printf("GetTcpInfo failed: %+v", err)
		} else {
			tcpInfoJson := NewTCPInfoJson(tcpInfo)
			footerJson, _ := json.Marshal(tcpInfoJson)
			copy(footerBuffer[1:], footerJson)
		}
	}

	if _, err := w.Write(footerBuffer); err != nil {
		log.Printf("write failed 2: %+v", err)
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	tcpCtx := GetTcpCtx(r.Context())

	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		return
	}

	// Read upload data
	buffer := make([]byte, 1024)
	totalBytes := 0
	for {
		n, err := r.Body.Read(buffer)
		totalBytes += n
		if err != nil {
			break
		}
	}

	log.Printf("Received %d bytes", totalBytes)

	var sendData []byte
	tcpInfo, err := tcpinfo.GetTcpInfo(tcpCtx.NativeConn)
	if err != nil {
		log.Printf("GetTcpInfo failed: %+v", err)
	} else {
		tcpInfoJson := NewTCPInfoJson(tcpInfo)
		sendData, _ = json.Marshal(tcpInfoJson)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(sendData)))
	if tcpCtx != nil {
		w.Header().Set("Connection", "close")
	}
	w.WriteHeader(200)
	_, _ = w.Write(sendData)
}

func writeJson(w http.ResponseWriter, data interface{}) {
	sendData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("ERROR: %+v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(sendData)))
	w.WriteHeader(200)
	_, _ = w.Write(sendData)
}

func getEnvAsInt(name string, def int) int {
	v := os.Getenv(name)
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("parse int failed: input=%s: %+v", v, err)
		return def
	}
	return i
}

func main() {
	var port int = getEnvAsInt("PORT", 3000)
	var quicPort int = -1
	var certFile string
	var keyFile string
	var generateCert bool
	var cacheDir string
	flag.IntVar(&port, "port", port, "listen port")
	flag.IntVar(&quicPort, "quic", -1, "enable quic server (0 is same to listen port)")
	flag.StringVar(&certFile, "cert", "", "TLS certificate file")
	flag.StringVar(&keyFile, "key", "", "TLS private key file")
	flag.StringVar(&cacheDir, "cache", "", "cert cache directory")
	flag.BoolVar(&generateCert, "generate-cert", false, "Generate self-signed certificate")
	flag.Parse()

	var err error

	slog.SetLogLoggerLevel(slog.LevelDebug)

	// Create custom server with TCP info collection
	mux := http.NewServeMux()
	mux.HandleFunc("/api/downloading", downloadHandler)
	mux.HandleFunc("/api/uploading", uploadHandler)

	var spkiList []string
	if quicPort >= 0 {
		if quicPort == 0 {
			quicPort = port
		}

		var tlsCert tls.Certificate

		if generateCert {
			_ = os.MkdirAll(cacheDir, 0700)

			keyFile = filepath.Join(cacheDir, "key.pem")
			certFile = filepath.Join(cacheDir, "cert.pem")

			if _, err = os.Stat(keyFile); errors.Is(err, os.ErrNotExist) {
				tlsCert, err = certutil.GenerateSelfSignedCert()
				if err != nil {
					log.Fatal("Failed to generate certificate:", err)
				}
				privateKeyDer, err := x509.MarshalPKCS8PrivateKey(tlsCert.PrivateKey)
				if err != nil {
					log.Fatal("Failed to marshal private key:", err)
				}
				privateKeyPem := &pem.Block{
					Type:  "PRIVATE KEY",
					Bytes: privateKeyDer,
				}
				err = os.WriteFile(keyFile, pem.EncodeToMemory(privateKeyPem), 0600)
				if err != nil {
					log.Fatal("Failed to write private key:", err)
				}
				certPem := &pem.Block{
					Type:  "CERTIFICATE",
					Bytes: tlsCert.Certificate[0],
				}
				err = os.WriteFile(certFile, pem.EncodeToMemory(certPem), 0600)
				if err != nil {
					log.Fatal("Failed to write certificate:", err)
				}
			}
		}

		tlsCert, err = tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			log.Fatal("Failed to load certificate:", err)
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
		}
		for i, bytes := range tlsCert.Certificate {
			spki, err := certutil.GetSpkiHashFromCertDer(bytes)
			if err != nil {
				log.Fatal("Failed to get SPKI hash:", err)
			}
			spkiList = append(spkiList, spki)
			log.Printf("SPKI[%d] HASH: %s", i, spki)
		}

		// HTTP/3 (QUIC) 서버
		quicServer := &http3.Server{
			Addr:      fmt.Sprintf(":%d", quicPort),
			Handler:   mux,
			TLSConfig: http3.ConfigureTLSConfig(tlsConfig),
			QUICConfig: &quic.Config{
				Tracer: qlog.DefaultConnectionTracer,
			},
		}

		headers := make(http.Header)
		_ = quicServer.SetQUICHeaders(headers)
		log.Printf("quic headers : %+v", headers)

		go func() {
			log.Printf("Starting HTTP/3 (QUIC) server on %s", quicServer.Addr)
			if err := quicServer.ListenAndServe(); err != nil {
				log.Fatal("QUIC server error:", err)
			}
		}()
	}

	mux.HandleFunc("/api/spki", func(writer http.ResponseWriter, request *http.Request) {
		writeJson(writer, spkiList)
	})

	// Serve embedded frontend directory
	frontendFS, err := fs.Sub(frontendFiles, "frontend")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/", http.FileServer(http.FS(frontendFS)))

	collector := &tcpInfoCollector{handler: mux}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: collector,
	}

	log.Printf("Server starting on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
