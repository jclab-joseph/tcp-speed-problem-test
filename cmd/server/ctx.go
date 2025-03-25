package main

import (
	"context"
	"net"
)

type AppCtx struct {
	NativeConn net.Conn
}

func GetAppCtx(ctx context.Context) *AppCtx {
	return ctx.Value("appCtx").(*AppCtx)
}

func WithAppCtx(ctx context.Context) (context.Context, *AppCtx) {
	v := &AppCtx{}
	return context.WithValue(ctx, "appCtx", v), v
}
