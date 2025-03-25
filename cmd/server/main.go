package main

import (
	crand "crypto/rand"
	"embed"
	"encoding/json"
	"github.com/jclab-joseph/tcp-speed-problem-test/pkg/tcpinfo"
	"io/fs"
	"log"
	randv2 "math/rand/v2"
	"net/http"
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
	reqCtx, appCtx := WithAppCtx(r.Context())
	appCtx.NativeConn = conn
	t.handler.ServeHTTP(newWriter, r.WithContext(reqCtx))
	newWriter.Flush()
	_ = conn.Close()
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	appCtx := GetAppCtx(r.Context())
	_ = appCtx

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
	w.Header().Set("Connection", "close")
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
			log.Printf("write failed: %+v", err)
			break
		}

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}

	footerBuffer := make([]byte, footerSize)

	tcpInfo, err := tcpinfo.GetTcpInfo(appCtx.NativeConn)
	if err != nil {
		log.Printf("GetTcpInfo failed: %+v", err)
	} else {
		tcpInfoJson := NewTCPInfoJson(tcpInfo)
		footerJson, _ := json.Marshal(tcpInfoJson)
		copy(footerBuffer[1:], footerJson)
	}

	if _, err := w.Write(footerBuffer); err != nil {
		log.Printf("write failed: %+v", err)
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	appCtx := GetAppCtx(r.Context())
	_ = appCtx

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
	tcpInfo, err := tcpinfo.GetTcpInfo(appCtx.NativeConn)
	if err != nil {
		log.Printf("GetTcpInfo failed: %+v", err)
	} else {
		tcpInfoJson := NewTCPInfoJson(tcpInfo)
		sendData, _ = json.Marshal(tcpInfoJson)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(sendData)))
	w.Header().Set("Connection", "close")
	w.WriteHeader(200)
	_, _ = w.Write(sendData)
}

func main() {
	// Create custom server with TCP info collection
	mux := http.NewServeMux()
	mux.HandleFunc("/api/downloading", downloadHandler)
	mux.HandleFunc("/api/uploading", uploadHandler)

	// Serve embedded frontend directory
	frontendFS, err := fs.Sub(frontendFiles, "frontend")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/", http.FileServer(http.FS(frontendFS)))

	collector := &tcpInfoCollector{handler: mux}

	server := &http.Server{
		Addr:    ":3000",
		Handler: collector,
	}

	log.Printf("Server starting on :3000")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
