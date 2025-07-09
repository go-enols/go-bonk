package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"

	raydium_launchpad "main/bonk"

	binary "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/go-enols/go-log"
	"github.com/go-enols/gosolana/ws"
)

// InitializeMonitor Initialize交易监听器
type InitializeMonitor struct {
	client   *rpc.Client
	wsClient *ws.Client
	ctx      context.Context
}

// NewInitializeMonitor 创建新的Initialize监听器
func NewInitializeMonitor(ctx context.Context, client *rpc.Client, wsClient *ws.Client) *InitializeMonitor {
	return &InitializeMonitor{
		client:   client,
		wsClient: wsClient,
		ctx:      ctx,
	}
}

// Start 开始监听Initialize交易
func (m *InitializeMonitor) Start() error {
	monitAddress := raydium_launchpad.ProgramID
	log.Info("开始监听程序地址:", monitAddress.String())

	// 订阅日志
	logs, err := m.wsClient.LogsSubscribeMentions(
		monitAddress,
		rpc.CommitmentConfirmed,
	)
	if err != nil {
		return fmt.Errorf("订阅日志失败: %w", err)
	}

	log.Info("成功订阅日志，开始监听...")

	for {
		select {
		case <-m.ctx.Done():
			return m.ctx.Err()
		default:
			// 接收日志数据
			data, err := logs.Recv(m.ctx)
			if err != nil {
				log.Error("接收日志失败:", err)
				continue
			}

			// 先检查日志是否包含Initialize指令
			if !m.containsInitializeInstruction(data.Value.Logs) {
				continue // 跳过不包含Initialize指令的交易
			}

			// 处理交易
			if err := m.processTransaction(data.Value.Signature); err != nil {
				log.Error("处理交易失败:", err)
			}
		}
	}
}

// containsInitializeInstruction 检查日志是否包含Initialize指令
func (m *InitializeMonitor) containsInitializeInstruction(logs []string) bool {
	// 检查每条日志消息
	for _, logMsg := range logs {
		// 示例检查应该有更详细的筛选以确保不会浪费带宽
		if bytes.Contains([]byte(logMsg), []byte("create")) {
			return true
		}
	}

	return false
}

// processTransaction 处理单个交易
func (m *InitializeMonitor) processTransaction(signature solana.Signature) error {
	// 获取完整交易信息
	transaction, err := m.client.GetTransaction(m.ctx, signature, &rpc.GetTransactionOpts{
		Commitment:                     rpc.CommitmentConfirmed,
		MaxSupportedTransactionVersion: &Verison,
	})
	if err != nil {
		return fmt.Errorf("获取交易失败: %w", err)
	}

	transactionInfo, err := transaction.Transaction.GetTransaction()
	if err != nil {
		return fmt.Errorf("解析交易失败: %w", err)
	}

	// 检查交易中的指令
	for i, instruction := range transactionInfo.Message.Instructions {
		if m.isInitializeInstruction(instruction, transactionInfo) {
			log.Info(fmt.Sprintf("发现Initialize交易! 签名: %s, 指令索引: %d", signature, i))
			m.handleInitializeInstruction(signature, instruction, transactionInfo)
		}
	}

	return nil
}

// isInitializeInstruction 检查是否是Initialize指令
func (m *InitializeMonitor) isInitializeInstruction(instruction solana.CompiledInstruction, transaction *solana.Transaction) bool {
	// 检查程序ID
	programIndex := instruction.ProgramIDIndex
	if int(programIndex) >= len(transaction.Message.AccountKeys) {
		return false
	}
	programID := transaction.Message.AccountKeys[programIndex]
	if !programID.Equals(raydium_launchpad.ProgramID) {
		return false
	}

	// 检查discriminator
	if len(instruction.Data) < 8 {
		return false
	}
	discriminator := instruction.Data[:8]
	return bytes.Equal(discriminator, raydium_launchpad.Instruction_Initialize[:])
}

// handleInitializeInstruction 处理Initialize指令
func (m *InitializeMonitor) handleInitializeInstruction(signature solana.Signature, instruction solana.CompiledInstruction, transaction *solana.Transaction) {
	log.Info("=== Initialize交易详情 ===")
	log.Info("交易签名:", signature.String())
	log.Info("Discriminator:", hex.EncodeToString(instruction.Data[:8]))
	log.Info("指令数据长度:", len(instruction.Data))
	log.Info("账户数量:", len(instruction.Accounts))

	// 解析指令参数
	if err := m.parseInitializeParams(instruction.Data[8:]); err != nil {
		log.Error("解析Initialize参数失败:", err)
	}

	// 解析账户信息
	m.parseInitializeAccounts(instruction, transaction)

	log.Info("=== 处理完成 ===")
}

// parseInitializeParams 解析Initialize指令参数
func (m *InitializeMonitor) parseInitializeParams(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("指令数据为空")
	}

	decoder := binary.NewBorshDecoder(data)

	// 根据Initialize指令的参数结构解析
	// baseMintParamParam MintParams
	var mintParams raydium_launchpad.MintParams
	if err := decoder.Decode(&mintParams); err != nil {
		return fmt.Errorf("解析MintParams失败: %w", err)
	}
	log.Info("MintParams:")
	log.Info("  Name:", mintParams.Name)
	log.Info("  Symbol:", mintParams.Symbol)
	log.Info("  Uri:", mintParams.Uri)
	log.Info("  Decimals:", mintParams.Decimals)

	// curveParamParam CurveParams
	var curveParams raydium_launchpad.CurveParams
	if err := decoder.Decode(&curveParams); err != nil {
		return fmt.Errorf("解析CurveParams失败: %w", err)
	}
	log.Info("CurveParams解析完成")

	// vestingParamParam VestingParams
	var vestingParams raydium_launchpad.VestingParams
	if err := decoder.Decode(&vestingParams); err != nil {
		return fmt.Errorf("解析VestingParams失败: %w", err)
	}
	log.Info("VestingParams:")
	log.Info("  TotalLockedAmount:", vestingParams.TotalLockedAmount)
	log.Info("  CliffPeriod:", vestingParams.CliffPeriod)
	log.Info("  UnlockPeriod:", vestingParams.UnlockPeriod)

	return nil
}

// parseInitializeAccounts 解析Initialize指令的账户信息
func (m *InitializeMonitor) parseInitializeAccounts(instruction solana.CompiledInstruction, transaction *solana.Transaction) {
	log.Info("=== 账户信息 ===")

	// Initialize指令的账户顺序（根据生成的代码）:
	accountNames := []string{
		"payer",               // 0
		"creator",             // 1
		"global_config",       // 2
		"platform_config",     // 3
		"authority",           // 4
		"pool_state",          // 5
		"base_mint",           // 6
		"quote_mint",          // 7
		"base_vault",          // 8
		"quote_vault",         // 9
		"metadata_account",    // 10
		"base_token_program",  // 11
		"quote_token_program", // 12
		"metadata_program",    // 13
		"system_program",      // 14
		"rent_program",        // 15
		"event_authority",     // 16
		"program",             // 17
	}

	for i, accountIndex := range instruction.Accounts {
		if int(accountIndex) < len(transaction.Message.AccountKeys) {
			accountKey := transaction.Message.AccountKeys[accountIndex]
			accountName := "unknown"
			if i < len(accountNames) {
				accountName = accountNames[i]
			}
			log.Info(fmt.Sprintf("  [%d] %s: %s", i, accountName, accountKey.String()))
		}
	}
}
