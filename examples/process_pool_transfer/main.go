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
	NetWork = rpc.Cluster{
		RPC: "https://mainnet.helius-rpc.com/?api-key=ce5ee933-a6ba-46b3-8e00-3f08bb2c49b1",
		WS:  "wss://mainnet.helius-rpc.com/?api-key=ce5ee933-a6ba-46b3-8e00-3f08bb2c49b1",
	}
	Verison uint64 = 1
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

	sign := solana.MustSignatureFromBase58("3RcdJKBLhq5ugyVn6Ygwvq4nvgu8arYtzoXvR7wtqHpVgsYAJBv9pa9gZkGyhLbKvbTuPSxyhu8YVH2sTNFLde18")

	data, err := poolMonitClient.ProcessTransaction(sign)
	if err != nil {
		log.Fatal(err)
	}
	log.Info(data)
}
