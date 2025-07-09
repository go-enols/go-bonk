package main

import (
	"context"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/go-enols/go-log"
	"github.com/go-enols/gosolana"
)

var (
	NetWork rpc.Cluster = rpc.MainNetBeta
	Verison uint64      = 1
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	option := gosolana.Option{
		RpcUrl: NetWork.RPC,
		WsUrl:  NetWork.WS,
	}

	wallet, err := gosolana.NewWallet(ctx, option)
	if err != nil {
		log.Fatal("创建钱包失败:", err)
	}

	client := wallet.GetClient()
	wsClient := wallet.GetWsClient()

	// 创建Initialize监听器
	monitor := NewInitializeMonitor(ctx, client, wsClient)

	// 示例解析一下创建交易的
	tx, err := solana.SignatureFromBase58("3RcdJKBLhq5ugyVn6Ygwvq4nvgu8arYtzoXvR7wtqHpVgsYAJBv9pa9gZkGyhLbKvbTuPSxyhu8YVH2sTNFLde18")
	if err != nil {
		log.Fatal(err)
	}
	monitor.processTransaction(tx)
}
