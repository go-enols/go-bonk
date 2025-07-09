# Go-Bonk: Raydium Launchpad Initialize 交易监听器

## 注意这里只是简单的示例代码,可能后续用到的时候我会进行后续更改

一个用于监听和解析 Raydium Launchpad Initialize 交易的 Go 语言工具。该工具通过 WebSocket 实时监听 Solana 区块链上的交易，并自动识别和解析 Initialize 指令。

## 🚀 特性

- **实时监听**: 通过 WebSocket 连接实时监听 Raydium Launchpad 程序的交易
- **智能过滤**: 在日志级别预过滤交易，只处理包含 Initialize 指令的交易，大幅减少 RPC 调用
- **详细解析**: 完整解析 Initialize 指令的参数和账户信息
- **高性能**: 优化的代码结构，避免不必要的反射调用
- **代理支持**: 支持 HTTP 和 WebSocket 代理配置

## 📦 项目结构

```
├── main.go                 # 主程序入口
├── initialize_monitor.go   # Initialize 交易监听器核心逻辑
├── bonk/                   # 生成的 Solana 程序绑定
│   ├── accounts.go         # 账户类型定义
│   ├── discriminators.go   # 指令和事件判别器
│   ├── instructions.go     # 指令构建器
│   ├── types.go           # 数据类型定义
│   └── ...
├── go.mod                  # Go 模块依赖
├── go.sum                  # 依赖校验和
└── README.md              # 项目文档
```

## 🛠️ 安装和使用

### 前置要求

- Go 1.24.4 或更高版本
- 网络连接（用于访问 Solana RPC 节点）

### 安装依赖

```bash
go mod download
```

### 配置

在 `main.go` 中修改网络设置：

```go
var (
    NetWork = rpc.Cluster{
        RPC: "https://mainnet.helius-rpc.com/?api-key=YOUR_API_KEY",
        WS:  "wss://mainnet.helius-rpc.com/?api-key=YOUR_API_KEY",
    }
    Verison uint64 = 1
)
```

### 代理配置（可选）

如果需要使用代理，可以在 `main.go` 中配置：

```go
option := gosolana.Option{
    RpcUrl:  NetWork.RPC,
    WsUrl:   NetWork.WS,
    Proxy:   "http://127.0.0.1:7890",    // HTTP 代理
    WsProxy: "http://127.0.0.1:7890",    // WebSocket 代理
}
```

### 运行

```bash
# 编译
go build .

# 运行
./main.exe
# 或者
go run .
```

## 📊 监听的信息

### Initialize 指令参数

- **MintParams**: 代币铸造参数
  - Name: 代币名称
  - Symbol: 代币符号
  - Uri: 元数据 URI
  - Decimals: 小数位数

- **CurveParams**: 价格曲线参数
  - 曲线类型（Linear/Fixed）
  - 相关参数

- **VestingParams**: 锁仓参数
  - TotalLockedAmount: 总锁仓数量
  - CliffPeriod: 锁仓期
  - UnlockPeriod: 解锁期

### 账户信息

- Payer: 支付账户
- Creator: 创建者账户
- Pool State: 池状态账户
- Base Mint: 基础代币账户
- Quote Mint: 报价代币账户
- 以及其他相关账户

## 🔧 核心功能

### InitializeMonitor

主要的监听器类，提供以下功能：

- `Start()`: 开始监听 Initialize 交易
- `containsInitializeInstruction()`: 预过滤日志，检查是否包含 Initialize 指令
- `processTransaction()`: 处理单个交易
- `handleInitializeInstruction()`: 解析 Initialize 指令详情
- `parseInitializeParams()`: 解析指令参数
- `parseInitializeAccounts()`: 解析账户信息

### 性能优化

- **日志预过滤**: 在处理交易前先检查日志是否包含 Initialize 指令的 discriminator
- **直接方法调用**: 移除反射调用，使用直接的方法调用提高性能
- **错误处理**: 完善的错误处理机制，确保程序稳定运行

## 📝 日志输出示例

```
[INFO] 开始监听程序地址: 6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P
[INFO] 成功订阅日志，开始监听...
[INFO] === Initialize交易详情 ===
[INFO] 交易签名: 5J7...abc
[INFO] Discriminator: afaf6d1f0d989bed
[INFO] 指令数据长度: 256
[INFO] 账户数量: 15
[INFO] === MintParams ===
[INFO] 代币名称: Example Token
[INFO] 代币符号: EXT
[INFO] 元数据URI: https://example.com/metadata.json
[INFO] 小数位数: 9
[INFO] === CurveParams ===
[INFO] 曲线类型: Linear
[INFO] === VestingParams ===
[INFO] 总锁仓数量: 1000000000
[INFO] 锁仓期: 86400
[INFO] 解锁期: 2592000
[INFO] === 账户信息 ===
[INFO] Payer: 7xK...def
[INFO] Creator: 9mN...ghi
[INFO] === 处理完成 ===
```

## 🔍 自定义配置

### 修改监听的程序地址

```go
// 在 initialize_monitor.go 中修改
monitAddress := raydium_launchpad.ProgramID
```

### 修改提交级别

```go
// 在 Start() 方法中修改
logs, err := m.wsClient.LogsSubscribeMentions(
    monitAddress,
    rpc.CommitmentConfirmed, // 可改为 rpc.CommitmentFinalized
)
```

## ⚠️ 注意事项

1. **RPC 限制**: 某些 RPC 提供商可能有请求频率限制，建议使用付费的 RPC 服务
2. **网络稳定性**: 确保网络连接稳定，避免 WebSocket 连接中断
3. **错误处理**: 程序包含完善的错误处理，但建议在生产环境中添加重连机制
4. **资源使用**: 长时间运行可能消耗较多内存，建议定期重启

## 📚 依赖项

- `github.com/gagliardetto/solana-go`: Solana Go SDK
- `github.com/go-enols/gosolana`: Solana 钱包和客户端封装
- `github.com/go-enols/go-log`: 日志库
- `github.com/gagliardetto/anchor-go`: Anchor 程序绑定生成器
- `github.com/gagliardetto/binary`: 二进制序列化库

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License

## 🔗 相关链接

- [Solana 官方文档](https://docs.solana.com/)
- [Raydium 官方网站](https://raydium.io/)
- [Anchor 框架](https://www.anchor-lang.com/)

---

**注意**: 这是一个用于学习和开发目的的工具，请在使用前充分测试，并遵守相关法律法规。