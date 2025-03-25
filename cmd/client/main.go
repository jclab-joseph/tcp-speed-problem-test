package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/jclab-joseph/tcp-speed-problem-test/pkg/tcpinfo"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"time"
)

type TCPInfoV0 struct {
	State             uint32
	Mss               uint32
	ConnectionTimeMs  uint64
	TimestampsEnabled bool
	RttUs             uint32
	MinRttUs          uint32
	BytesInFlight     uint32
	Cwnd              uint32
	SndWnd            uint32
	RcvWnd            uint32
	RcvBuf            uint32
	BytesOut          uint64
	BytesIn           uint64
	BytesReordered    uint32
	BytesRetrans      uint32
	FastRetrans       uint32
	DupAcksIn         uint32
	TimeoutEpisodes   uint32
	SynRetrans        uint8
}

type RawJson = map[string]interface{}

func consumeBuffer(r io.Reader) (int, RawJson, error) {
	var tcpInfo RawJson

	var buffer [4096]byte
	var totalBytes int
	var lastChunk []byte // 마지막 청크를 저장할 변수

	for {
		n, err := r.Read(buffer[:])
		totalBytes += n

		// 마지막 4KB를 유지
		if len(lastChunk)+n > 4096 {
			lastChunk = append(lastChunk[len(lastChunk)+n-4096:], buffer[:n]...)
		} else {
			lastChunk = append(lastChunk, buffer[:n]...)
		}

		if err != nil {
			if err == io.EOF {
				// JSON 데이터 추출
				if len(lastChunk) > 0 {
					// JSON 시작 위치 찾기 ('{' 문자)
					jsonStartIndex := -1
					for i, b := range lastChunk {
						if b == '{' {
							jsonStartIndex = i
							break
						}
					}

					// JSON 끝 위치 찾기 (첫 번째 null 바이트)
					jsonEndIndex := -1
					if jsonStartIndex != -1 {
						for i := jsonStartIndex; i < len(lastChunk); i++ {
							if lastChunk[i] == 0 {
								jsonEndIndex = i
								break
							}
						}
					}

					// JSON 파싱
					if jsonStartIndex != -1 && jsonEndIndex != -1 {
						jsonData := lastChunk[jsonStartIndex:jsonEndIndex]
						if err := json.Unmarshal(jsonData, &tcpInfo); err != nil {
							log.Printf("json unmarshal failed: %+v", err)
						}
					}
				}
				break
			}
			log.Printf("Error reading response body: %+v", err)
			return totalBytes, tcpInfo, err
		}
	}

	return totalBytes, tcpInfo, nil
}

type StatCtx struct {
	TCPInfoV0 interface{}
}

func httpGetAndMeasureSpeed(client *http.Client, baseUrl *url.URL) float64 {
	var targetUrl url.URL = *baseUrl
	targetUrl.Query().Set("n", fmt.Sprintf("%f", rand.Float32()))

	stat := &StatCtx{}
	ctx := context.WithValue(context.Background(), "stat", stat)

	req, _ := http.NewRequestWithContext(ctx, "GET", targetUrl.String(), nil)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("HTTP Get failed: %+v", err)
		return -1
	}

	startTime := time.Now()
	totalBytes, jsonOut, err := consumeBuffer(resp.Body)
	elapsedTime := time.Since(startTime).Seconds()

	// 초당 비트 수(bps) 계산
	bps := float64(totalBytes*8) / elapsedTime

	log.Printf("Download speed: %.2f Mbps (%.2f bytes in %.2f seconds)",
		bps/1000000, float64(totalBytes), elapsedTime)

	_ = resp.Body.Close()

	if jsonOut != nil {
		log.Printf("Server Side Result:")
		raw, _ := json.MarshalIndent(jsonOut, "", "  ")
		fmt.Println(string(raw))
	}
	log.Printf("Client Side Result:")
	printStat(stat.TCPInfoV0)

	return bps
}

func main() {
	var targetUrl string
	var iteration int
	flag.StringVar(&targetUrl, "url", "http://127.0.0.1:3000/api/downloading?size=1", "")
	flag.IntVar(&iteration, "iter", 3, "")
	flag.Parse()

	parsedUrl, err := url.Parse(targetUrl)
	if err != nil {
		log.Fatalf("invalid targetUrl: %+v", err)
		return
	}

	sysDialer := &net.Dialer{}
	httpTransport := &http.Transport{
		DisableKeepAlives: true,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			stat := ctx.Value("stat")

			conn, err := sysDialer.DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}

			tcpConn, ok := conn.(*net.TCPConn)
			if !ok {
				fmt.Println("TCP 연결로 변환할 수 없습니다")
				return conn, nil
			}

			wc := &wrappedConn{
				conn: tcpConn,
				stat: stat.(*StatCtx),
			}
			log.Printf("TCP Connected to %+v", tcpConn.RemoteAddr())
			return wc, nil
		},
	}
	httpClient := &http.Client{
		Transport: httpTransport,
	}

	var total float64
	for i := 0; i < iteration; i++ {
		bps := httpGetAndMeasureSpeed(httpClient, parsedUrl)
		if bps > 0 {
			total += bps
		}
		time.Sleep(time.Microsecond * 250)
	}
	total /= float64(iteration)

	log.Printf("Average bps: %f Mbps", total/1000000)
}

type wrappedConn struct {
	conn *net.TCPConn
	stat *StatCtx
}

func (w *wrappedConn) Read(b []byte) (n int, err error) {
	return w.conn.Read(b)
}

func (w *wrappedConn) Write(b []byte) (n int, err error) {
	return w.conn.Write(b)
}

func (w *wrappedConn) Close() error {
	info, err := tcpinfo.GetTcpInfo(w.conn)
	if err != nil {
		fmt.Println("TCP 정보 가져오기 실패:", err)
	} else {
		w.stat.TCPInfoV0 = info
	}

	return w.conn.Close()
}

func (w *wrappedConn) LocalAddr() net.Addr {
	return w.conn.LocalAddr()
}

func (w *wrappedConn) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}

func (w *wrappedConn) SetDeadline(t time.Time) error {
	return w.conn.SetDeadline(t)
}

func (w *wrappedConn) SetReadDeadline(t time.Time) error {
	return w.conn.SetReadDeadline(t)
}

func (w *wrappedConn) SetWriteDeadline(t time.Time) error {
	return w.conn.SetWriteDeadline(t)
}

func printStat(input interface{}) {
	info := input.(*tcpinfo.TCPInfoV0)
	// 재전송 관련 정보 출력
	fmt.Printf("\tMSS: %d\n", info.Mss)
	fmt.Printf("\tConnection Time: %d ms\n", info.ConnectionTimeMs)
	fmt.Printf("\tTimestamps Enabled: %v\n", info.TimestampsEnabled)
	fmt.Printf("\tRTT: %d  us\n", info.RttUs)
	fmt.Printf("\tMin RTT: %d us\n", info.MinRttUs)
	fmt.Printf("\tBytes In Flight: %d\n", info.BytesInFlight)
	fmt.Printf("\tCongestion Window (cwnd): %d\n", info.Cwnd)
	fmt.Printf("\tSend Window (sndwnd): %d\n", info.SndWnd)
	fmt.Printf("\tReceive Window (rcvwnd): %d\n", info.RcvWnd)
	fmt.Printf("\tReceive Buffer (rcvbuf): %d\n", info.RcvBuf)
	fmt.Printf("\tBytes Sent: %d\n", info.BytesOut)
	fmt.Printf("\tBytes Received: %d\n", info.BytesIn)
	fmt.Printf("\tBytes Reordered: %d\n", info.BytesReordered)
	fmt.Printf("\tBytes Retransmitted: %d\n", info.BytesRetrans)
	fmt.Printf("\tFast Retransmits: %d\n", info.FastRetrans)
	fmt.Printf("\tDuplicate ACKs In: %d\n", info.DupAcksIn)
	fmt.Printf("\tTimeout Episodes: %d\n", info.TimeoutEpisodes)
	fmt.Printf("\tSYN Retransmits: %d\n", info.SynRetrans)
}
