# Parking Operations

这是一个纯 Go 的停车入口/出口示例。入口使用 HMAC-SHA256 签发可离线校验的凭证，凭证载荷包含车牌、UTC 入场时间和区域码；出口只需共享密钥即可校验签名并输出收费前状态。审计数据保存在进程内存中。

## 环境

- Go 1.25.13
- `GOTOOLCHAIN=local`
- 无第三方依赖，兼容 `CGO_ENABLED=0`

```sh
export GOTOOLCHAIN=local
go test -count=1 ./...
```

业务链路回归测试会稳定暴露当前注入缺陷：有效凭证返回 `valid`，但对应审计状态不一致。

## 示例入口

运行固定夹具的完整入口、出口和审计流程：

```sh
GOTOOLCHAIN=local go run ./cmd/parking-example demo
```

单独签发入口凭证：

```sh
GOTOOLCHAIN=local go run ./cmd/parking-example entry \
  -plate A-12345 \
  -time 2026-08-16T08:30:00+08:00 \
  -zone P2
```

将签发结果中的 `token` 传给出口：

```sh
GOTOOLCHAIN=local go run ./cmd/parking-example exit -credential '<token>'
```

库式调用从模块根包 `parkingops` 导入，依次创建 `Issuer`、`MemoryAuditRepository` 和 `ExitValidator`。示例密钥仅用于本地固定夹具，生产环境必须由部署系统提供独立密钥。
