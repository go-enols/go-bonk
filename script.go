package bonk

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	raydium_launchpad "github.com/go-enols/go-bonk/idl"

	binary "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/go-enols/go-log"
	"github.com/go-enols/gosolana"
	"github.com/go-enols/gosolana/ws"
)

var Verison uint64 = 0

// InitializeTransactionData Initialize交易解析后的汇总数据
type InitializeAccounts struct {
	Payer             solana.PublicKey                  `json:"payer"`
	Creator           solana.PublicKey                  `json:"creator"`
	GlobalConfig      *raydium_launchpad.GlobalConfig   `json:"global_config,omitempty"`
	PlatformConfig    *raydium_launchpad.PlatformConfig `json:"platform_config,omitempty"`
	Authority         solana.PublicKey                  `json:"authority"`
	PoolState         *raydium_launchpad.PoolState      `json:"pool_state,omitempty"`
	BaseMint          solana.PublicKey                  `json:"base_mint"`
	QuoteMint         solana.PublicKey                  `json:"quote_mint"`
	BaseVault         solana.PublicKey                  `json:"base_vault"`
	QuoteVault        solana.PublicKey                  `json:"quote_vault"`
	MetadataAccount   solana.PublicKey                  `json:"metadata_account"`
	BaseTokenProgram  solana.PublicKey                  `json:"base_token_program"`
	QuoteTokenProgram solana.PublicKey                  `json:"quote_token_program"`
	MetadataProgram   solana.PublicKey                  `json:"metadata_program"`
	SystemProgram     solana.PublicKey                  `json:"system_program"`
	RentProgram       solana.PublicKey                  `json:"rent_program"`
	EventAuthority    solana.PublicKey                  `json:"event_authority"`
	Program           solana.PublicKey                  `json:"program"`
}

type InitializeTransactionData struct {
	Signature     string                          `json:"signature"`
	Discriminator string                          `json:"discriminator"`
	DataLength    int                             `json:"data_length"`
	AccountCount  int                             `json:"account_count"`
	MintParams    raydium_launchpad.MintParams    `json:"mint_params"` // 代币的元数据
	CurveParams   raydium_launchpad.CurveParams   `json:"curve_params"`
	VestingParams raydium_launchpad.VestingParams `json:"vesting_params"` // 包含解锁时间总供应量等信息
	Accounts      InitializeAccounts              `json:"accounts"`
	RawAccounts   map[string]string               `json:"raw_accounts"`
	TransferTime  time.Time                       `json:"transfer_time"`
}

// PoolMonit Initialize交易监听器
type PoolMonit struct {
	*gosolana.Wallet
	ctx context.Context
	Pip chan *InitializeTransactionData
}

func NewPoolMonit(ctx context.Context, option ...gosolana.Option) (*PoolMonit, error) {
	wallet, err := gosolana.NewWallet(ctx, option...)
	if err != nil {
		return nil, err
	}
	return &PoolMonit{
		Wallet: wallet,
		ctx:    ctx,
		Pip:    make(chan *InitializeTransactionData),
	}, nil
}

// containsInitializeInstruction 检查日志是否包含Initialize指令
func (p *PoolMonit) containsInitializeInstruction(logs []string) bool {
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
func (p *PoolMonit) ProcessTransaction(signature solana.Signature) (*InitializeTransactionData, error) {
	// 获取完整交易信息
	transaction, err := p.GetClient().GetTransaction(p.ctx, signature, &rpc.GetTransactionOpts{
		Commitment:                     rpc.CommitmentConfirmed,
		MaxSupportedTransactionVersion: &Verison,
	})
	if err != nil {
		return nil, fmt.Errorf("获取交易失败: %w", err)
	}

	transactionInfo, err := transaction.Transaction.GetTransaction()
	if err != nil {
		return nil, fmt.Errorf("解析交易失败: %w", err)
	}

	// 检查交易中的指令
	for i, instruction := range transactionInfo.Message.Instructions {
		if p.isInitializeInstruction(instruction, transactionInfo) {
			log.Info(fmt.Sprintf("发现Initialize交易! 签名: %s, 指令索引: %d", signature, i))
			txData := p.handleInitializeInstruction(signature, instruction, transactionInfo)
			txData.TransferTime = transaction.BlockTime.Time()
			return txData, nil
		}
	}

	return nil, errors.New("不是Initialize交易")

}

// isInitializeInstruction 检查是否是Initialize指令
func (p *PoolMonit) isInitializeInstruction(instruction solana.CompiledInstruction, transaction *solana.Transaction) bool {
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
func (p *PoolMonit) handleInitializeInstruction(signature solana.Signature, instruction solana.CompiledInstruction, transaction *solana.Transaction) *InitializeTransactionData {
	data := &InitializeTransactionData{
		Signature:     signature.String(),
		Discriminator: hex.EncodeToString(instruction.Data[:8]),
		DataLength:    len(instruction.Data),
		AccountCount:  len(instruction.Accounts),
		Accounts:      InitializeAccounts{},
		RawAccounts:   make(map[string]string),
	}

	// 解析指令参数
	if err := p.parseInitializeParams(instruction.Data[8:], data); err != nil {
		log.Error("解析Initialize参数失败:", err)
		return data
	}

	// 解析账户信息
	if err := p.parseInitializeAccounts(instruction, transaction, data); err != nil {
		log.Error("解析账户信息失败:", err)
	}

	log.Info("=== Initialize交易解析完成 ===")
	log.Info("交易签名:", data.Signature)
	log.Info("代币名称:", data.MintParams.Name)
	log.Info("代币符号:", data.MintParams.Symbol)

	return data
}

// parseInitializeParams 解析Initialize指令参数
func (p *PoolMonit) parseInitializeParams(rawData []byte, txData *InitializeTransactionData) error {
	if len(rawData) == 0 {
		return fmt.Errorf("指令数据为空")
	}

	decoder := binary.NewBorshDecoder(rawData)

	// 根据Initialize指令的参数结构解析
	// baseMintParamParam MintParams
	if err := decoder.Decode(&txData.MintParams); err != nil {
		return fmt.Errorf("解析MintParams失败: %w", err)
	}

	// curveParamParam CurveParams
	if err := decoder.Decode(&txData.CurveParams); err != nil {
		return fmt.Errorf("解析CurveParams失败: %w", err)
	}

	// vestingParamParam VestingParams
	if err := decoder.Decode(&txData.VestingParams); err != nil {
		return fmt.Errorf("解析VestingParams失败: %w", err)
	}

	return nil
}

// fetchAccountData 获取并解析账户数据
func (p *PoolMonit) fetchAccountData(txData *InitializeTransactionData) error {
	// 获取GlobalConfig账户数据
	globalConfigKey := txData.RawAccounts["global_config"]
	if globalConfigKey != "" {
		if globalConfigPubkey, err := solana.PublicKeyFromBase58(globalConfigKey); err == nil {
			globalConfigData, err := p.GetClient().GetAccountInfo(p.ctx, globalConfigPubkey)
			if err == nil && globalConfigData != nil && globalConfigData.Value != nil {
				if globalConfig, err := raydium_launchpad.ParseAccount_GlobalConfig(globalConfigData.Value.Data.GetBinary()); err == nil {
					txData.Accounts.GlobalConfig = globalConfig
				}
			}
		}
	}

	// 获取PlatformConfig账户数据
	platformConfigKey := txData.RawAccounts["platform_config"]
	if platformConfigKey != "" {
		if platformConfigPubkey, err := solana.PublicKeyFromBase58(platformConfigKey); err == nil {
			platformConfigData, err := p.GetClient().GetAccountInfo(p.ctx, platformConfigPubkey)
			if err == nil && platformConfigData != nil && platformConfigData.Value != nil {
				if platformConfig, err := raydium_launchpad.ParseAccount_PlatformConfig(platformConfigData.Value.Data.GetBinary()); err == nil {
					txData.Accounts.PlatformConfig = platformConfig
				}
			}
		}
	}

	// 获取PoolState账户数据
	poolStateKey := txData.RawAccounts["pool_state"]
	if poolStateKey != "" {
		if poolStatePubkey, err := solana.PublicKeyFromBase58(poolStateKey); err == nil {
			poolStateData, err := p.GetClient().GetAccountInfo(p.ctx, poolStatePubkey)
			if err == nil && poolStateData != nil && poolStateData.Value != nil {
				if poolState, err := raydium_launchpad.ParseAccount_PoolState(poolStateData.Value.Data.GetBinary()); err == nil {
					txData.Accounts.PoolState = poolState
				}
			}
		}
	}

	return nil
}

// parseInitializeAccounts 解析Initialize指令的账户信息
func (p *PoolMonit) parseInitializeAccounts(instruction solana.CompiledInstruction, transaction *solana.Transaction, txData *InitializeTransactionData) error {
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

	// 填充原始账户信息
	for i, accountIndex := range instruction.Accounts {
		if int(accountIndex) < len(transaction.Message.AccountKeys) {
			accountKey := transaction.Message.AccountKeys[accountIndex]
			accountName := "unknown"
			if i < len(accountNames) {
				accountName = accountNames[i]
			}
			txData.RawAccounts[accountName] = accountKey.String()
		}
	}

	// 填充结构化账户信息
	if len(instruction.Accounts) >= 18 {
		txData.Accounts.Payer = transaction.Message.AccountKeys[instruction.Accounts[0]]
		txData.Accounts.Creator = transaction.Message.AccountKeys[instruction.Accounts[1]]
		txData.Accounts.Authority = transaction.Message.AccountKeys[instruction.Accounts[4]]
		txData.Accounts.BaseMint = transaction.Message.AccountKeys[instruction.Accounts[6]]
		txData.Accounts.QuoteMint = transaction.Message.AccountKeys[instruction.Accounts[7]]
		txData.Accounts.BaseVault = transaction.Message.AccountKeys[instruction.Accounts[8]]
		txData.Accounts.QuoteVault = transaction.Message.AccountKeys[instruction.Accounts[9]]
		txData.Accounts.MetadataAccount = transaction.Message.AccountKeys[instruction.Accounts[10]]
		txData.Accounts.BaseTokenProgram = transaction.Message.AccountKeys[instruction.Accounts[11]]
		txData.Accounts.QuoteTokenProgram = transaction.Message.AccountKeys[instruction.Accounts[12]]
		txData.Accounts.MetadataProgram = solana.TokenMetadataProgramID
		txData.Accounts.SystemProgram = transaction.Message.AccountKeys[instruction.Accounts[14]]
		txData.Accounts.RentProgram = solana.SysVarRentPubkey
		txData.Accounts.EventAuthority = transaction.Message.AccountKeys[instruction.Accounts[16]]
		txData.Accounts.Program = transaction.Message.AccountKeys[instruction.Accounts[17]]

		// 尝试获取并解析账户数据
		if err := p.fetchAccountData(txData); err != nil {
			log.Error("获取账户数据失败:", err)
		}
	}

	return nil
}

// GetInitializeTransactionData 获取Initialize交易的解析数据
func (p *PoolMonit) GetInitializeTransactionData(signature solana.Signature) (*InitializeTransactionData, error) {
	// 获取完整交易信息
	transaction, err := p.GetClient().GetTransaction(p.ctx, signature, &rpc.GetTransactionOpts{
		Commitment:                     rpc.CommitmentConfirmed,
		MaxSupportedTransactionVersion: &Verison,
	})
	if err != nil {
		return nil, fmt.Errorf("获取交易失败: %w", err)
	}

	transactionInfo, err := transaction.Transaction.GetTransaction()
	if err != nil {
		return nil, fmt.Errorf("解析交易失败: %w", err)
	}

	// 检查交易中的指令
	for _, instruction := range transactionInfo.Message.Instructions {
		if p.isInitializeInstruction(instruction, transactionInfo) {
			return p.handleInitializeInstruction(signature, instruction, transactionInfo), nil
		}
	}

	return nil, fmt.Errorf("未找到Initialize指令")
}

// ProcessTransaction 处理WebSocket接收到的日志结果
func (p *PoolMonit) ProcessTransactionLogs(logResult *ws.LogResult) {
	if !p.containsInitializeInstruction(logResult.Value.Logs) {
		// 不是需要的交易直接抛弃
		return
	}
	data, err := p.ProcessTransaction(logResult.Value.Signature)
	if err != nil {
		log.Error("处理交易失败:", err)
		return
	}
	select {
	case <-time.After(time.Second * 3):
		log.Warningf("交易 %s 处理超时", data.Signature)
	case p.Pip <- data:
	}
}
