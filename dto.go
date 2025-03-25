package main

import "golang.org/x/sys/unix"

type TCPInfoJson struct {
	State                uint8  `json:"state"`
	Ca_state             uint8  `json:"caState"`
	Retransmits          uint8  `json:"retransmits"`
	Probes               uint8  `json:"probes"`
	Backoff              uint8  `json:"backoff"`
	Options              uint8  `json:"options"`
	Rto                  uint32 `json:"rto"`
	Ato                  uint32 `json:"ato"`
	Snd_mss              uint32 `json:"sndMss"`
	Rcv_mss              uint32 `json:"rcvMss"`
	Unacked              uint32 `json:"unacked"`
	Sacked               uint32 `json:"sacked"`
	Lost                 uint32 `json:"lost"`
	Retrans              uint32 `json:"retrans"`
	Fackets              uint32 `json:"fackets"`
	Last_data_sent       uint32 `json:"lastDataSent"`
	Last_ack_sent        uint32 `json:"lastAckSent"`
	Last_data_recv       uint32 `json:"lastDataRecv"`
	Last_ack_recv        uint32 `json:"lastAckRecv"`
	Pmtu                 uint32 `json:"pmtu"`
	Rcv_ssthresh         uint32 `json:"rcvSsthresh"`
	Rtt                  uint32 `json:"rtt"`
	Rttvar               uint32 `json:"rttvar"`
	Snd_ssthresh         uint32 `json:"sndSsthresh"`
	Snd_cwnd             uint32 `json:"sndCwnd"`
	Advmss               uint32 `json:"advmss"`
	Reordering           uint32 `json:"reordering"`
	Rcv_rtt              uint32 `json:"rcvRtt"`
	Rcv_space            uint32 `json:"rcvSpace"`
	Total_retrans        uint32 `json:"totalRetrans"`
	Pacing_rate          uint64 `json:"pacingRate"`
	Max_pacing_rate      uint64 `json:"maxPacingRate"`
	Bytes_acked          uint64 `json:"bytesAcked"`
	Bytes_received       uint64 `json:"bytesReceived"`
	Segs_out             uint32 `json:"segsOut"`
	Segs_in              uint32 `json:"segsIn"`
	Notsent_bytes        uint32 `json:"notsentBytes"`
	Min_rtt              uint32 `json:"minRtt"`
	Data_segs_in         uint32 `json:"dataSegsIn"`
	Data_segs_out        uint32 `json:"dataSegsOut"`
	Delivery_rate        uint64 `json:"deliveryRate"`
	Busy_time            uint64 `json:"busyTime"`
	Rwnd_limited         uint64 `json:"rwndLimited"`
	Sndbuf_limited       uint64 `json:"sndbufLimited"`
	Delivered            uint32 `json:"delivered"`
	Delivered_ce         uint32 `json:"deliveredCe"`
	Bytes_sent           uint64 `json:"bytesSent"`
	Bytes_retrans        uint64 `json:"bytesRetrans"`
	Dsack_dups           uint32 `json:"dsackDups"`
	Reord_seen           uint32 `json:"reordSeen"`
	Rcv_ooopack          uint32 `json:"rcvOoopack"`
	Snd_wnd              uint32 `json:"sndWnd"`
	Rcv_wnd              uint32 `json:"rcvWnd"`
	Rehash               uint32 `json:"rehash"`
	Total_rto            uint16 `json:"totalRto"`
	Total_rto_recoveries uint16 `json:"totalRtoRecoveries"`
	Total_rto_time       uint32 `json:"totalRtoTime"`
}

func NewTCPInfoJson(tcpInfo *unix.TCPInfo) *TCPInfoJson {
	return &TCPInfoJson{
		State:                tcpInfo.State,
		Ca_state:             tcpInfo.Ca_state,
		Retransmits:          tcpInfo.Retransmits,
		Probes:               tcpInfo.Probes,
		Backoff:              tcpInfo.Backoff,
		Options:              tcpInfo.Options,
		Rto:                  tcpInfo.Rto,
		Ato:                  tcpInfo.Ato,
		Snd_mss:              tcpInfo.Snd_mss,
		Rcv_mss:              tcpInfo.Rcv_mss,
		Unacked:              tcpInfo.Unacked,
		Sacked:               tcpInfo.Sacked,
		Lost:                 tcpInfo.Lost,
		Retrans:              tcpInfo.Retrans,
		Fackets:              tcpInfo.Fackets,
		Last_data_sent:       tcpInfo.Last_data_sent,
		Last_ack_sent:        tcpInfo.Last_ack_sent,
		Last_data_recv:       tcpInfo.Last_data_recv,
		Last_ack_recv:        tcpInfo.Last_ack_recv,
		Pmtu:                 tcpInfo.Pmtu,
		Rcv_ssthresh:         tcpInfo.Rcv_ssthresh,
		Rtt:                  tcpInfo.Rtt,
		Rttvar:               tcpInfo.Rttvar,
		Snd_ssthresh:         tcpInfo.Snd_ssthresh,
		Snd_cwnd:             tcpInfo.Snd_cwnd,
		Advmss:               tcpInfo.Advmss,
		Reordering:           tcpInfo.Reordering,
		Rcv_rtt:              tcpInfo.Rcv_rtt,
		Rcv_space:            tcpInfo.Rcv_space,
		Total_retrans:        tcpInfo.Total_retrans,
		Pacing_rate:          tcpInfo.Pacing_rate,
		Max_pacing_rate:      tcpInfo.Max_pacing_rate,
		Bytes_acked:          tcpInfo.Bytes_acked,
		Bytes_received:       tcpInfo.Bytes_received,
		Segs_out:             tcpInfo.Segs_out,
		Segs_in:              tcpInfo.Segs_in,
		Notsent_bytes:        tcpInfo.Notsent_bytes,
		Min_rtt:              tcpInfo.Min_rtt,
		Data_segs_in:         tcpInfo.Data_segs_in,
		Data_segs_out:        tcpInfo.Data_segs_out,
		Delivery_rate:        tcpInfo.Delivery_rate,
		Busy_time:            tcpInfo.Busy_time,
		Rwnd_limited:         tcpInfo.Rwnd_limited,
		Sndbuf_limited:       tcpInfo.Sndbuf_limited,
		Delivered:            tcpInfo.Delivered,
		Delivered_ce:         tcpInfo.Delivered_ce,
		Bytes_sent:           tcpInfo.Bytes_sent,
		Bytes_retrans:        tcpInfo.Bytes_retrans,
		Dsack_dups:           tcpInfo.Dsack_dups,
		Reord_seen:           tcpInfo.Reord_seen,
		Rcv_ooopack:          tcpInfo.Rcv_ooopack,
		Snd_wnd:              tcpInfo.Snd_wnd,
		Rcv_wnd:              tcpInfo.Rcv_wnd,
		Rehash:               tcpInfo.Rehash,
		Total_rto:            tcpInfo.Total_rto,
		Total_rto_recoveries: tcpInfo.Total_rto_recoveries,
		Total_rto_time:       tcpInfo.Total_rto_time,
	}
}
