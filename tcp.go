package main

import (
	"errors"
	"golang.org/x/sys/unix"
	"log"
	"net"
)

func GetTcpInfo(conn net.Conn) (*unix.TCPInfo, error) {
	// Get TCP info before closing
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		file, err := tcpConn.File()
		if err != nil {
			log.Printf("Error getting file descriptor: %v", err)
			return nil, err
		} else {
			info, err := unix.GetsockoptTCPInfo(int(file.Fd()), unix.IPPROTO_TCP, unix.TCP_INFO)
			if err != nil {
				log.Printf("Error getting TCP info: %v", err)
				return nil, err
			} else {
				return info, nil
				//log.Printf("TCP Connection closed with info: RTT: %d µs, Lost: %d, Retrans: %d, Rtt_var: %d",
				//	info.Rtt,
				//	info.Lost,
				//	info.Retrans,
				//	info.Rttvar)
			}
			file.Close()
		}
	}
	return nil, errors.New("no tcp conn")
}
