package main

import (
	"context"

	"github.com/go-enols/go-log"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/go-enols/go-bonk"
	"github.com/go-enols/gosolana"
)

var (
	NetWork rpc.Cluster = rpc.MainNetBeta
	Verison uint64      = 1
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 构建option复用内部的client以及wsclient
	opt := gosolana.NewDefaultOption(ctx, gosolana.Option{
		RpcUrl: NetWork.RPC,
		WsUrl:  NetWork.WS,
	})

	poolMonitClient, err := bonk.NewPoolMonit(ctx, opt)
	if err != nil {
		log.Fatal(err)
	}

	var ProgramID = solana.MustPublicKeyFromBase58("LanMV9sAd7wArD4vJFi2qDdfnVhFxYSUg6eADduJ3uj")

	monit := bonk.NewClient(ctx, opt)

	monit.UseLog(poolMonitClient.ProcessTransactionLogs) // 添加一个处理交易日志的中间件

	go monit.Start(ctx, ProgramID, rpc.CommitmentConfirmed)
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-poolMonitClient.Pip:
			log.Printf("交易 %s 处理成功", data.Signature)
			log.Info(data)
		}
	}
}
