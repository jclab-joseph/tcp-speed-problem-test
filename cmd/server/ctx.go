package main

import (
	"context"
	"net"
)

type TcpCtx struct {
	NativeConn net.Conn
}

func GetTcpCtx(ctx context.Context) *TcpCtx {
	v, ok := ctx.Value("tcpCtx").(*TcpCtx)
	if ok {
		return v
	}
	return nil
}

func WithTcpCtx(ctx context.Context) (context.Context, *TcpCtx) {
	v := &TcpCtx{}
	return context.WithValue(ctx, "tcpCtx", v), v
}
