# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 在此仓库中工作提供指导。

## 项目概述

pflag 是 Go 标准库 `flag` 包的直接替代品，实现 POSIX/GNU 风格的 `--flags`。它被广泛用作 [cobra](https://github.com/spf13/cobra) 的 flag 解析库。这是一个单 Go 包（`package pflag`），没有子包。

## 构建和测试命令

```bash
# 运行所有测试（CI 使用 race 检测器）
go test -race -v ./...

# 运行单个测试
go test -race -v -run TestXxx ./...

# Lint（CI 使用 golangci-lint v2.7）
golangci-lint run
```

## Go 版本兼容性

模块目标版本为 `go 1.12`。CI 测试覆盖 Go 1.12、1.21、1.22、1.23、oldstable 和 stable。不要使用 Go 1.12 中不可用的语言特性或标准库 API，除非通过 build tag 隔离（参考 `func_go1.21_test.go`、`bool_func_go1.21_test.go`）。因此非测试代码中不使用 `errors.Is`/`errors.As`。

## 架构

### 核心类型（`flag.go`）

- **`FlagSet`** — flag 集合；持有已定义（formal）和已设置（actual）的 flag map、shorthands、args 和解析状态。`CommandLine` 是默认的全局 `FlagSet`。
- **`Flag`** — 表示单个 flag：名称、shorthand、用法说明、值、默认值、废弃标记等。
- **`Value` 接口** — `String()`、`Set(string) error`、`Type() string`。每种 flag 类型都实现此接口。
- **`SliceValue` 接口** — 列表值 flag 的辅助接口：`Append`、`Replace`、`GetSlice`。

### Flag 类型文件模式

每种 flag 类型在独立文件中遵循一致的模式（如 `int.go`、`bool.go`、`string_slice.go`）：

1. 未导出的值类型（如 `type intValue int`）实现 `Value`
2. `newXxxValue(val, p)` 构造函数设置指针
3. `Set(string) error`、`Type() string`、`String() string` 方法
4. `getFlagType` 的转换函数（如 `intConv`）
5. `FlagSet.GetXxx(name)` — 获取类型化的值
6. `FlagSet.XxxVar`、`FlagSet.XxxVarP` — 绑定到已有变量
7. `FlagSet.Xxx`、`FlagSet.XxxP` — 分配并返回指针
8. 顶层便利函数委托给 `CommandLine`

### 错误类型（`errors.go`）

结构化错误类型：`NotExistError`、`ValueRequiredError`、`InvalidValueError`、`InvalidSyntaxError`。每个都有访问器方法用于编程式错误检查。

### Go flag 互操作（`golangflag.go`）

`AddGoFlagSet` / `AddGoFlag` 将 Go 的 `flag` 包桥接到 pflag。`CopyToGoFlagSet` 做反向操作。`ParseSkippedFlags` 处理 pflag 跳过的 `-test.*` flag。

### 未消费参数收集（`flag.go`、`unknown_flags_test.go`）

`FlagSet.unknownFlags` 记录解析过程中所有未被消费的参数——未知 flag、其被 strip 的值、位置参数、`--` 终止符、终止符后参数、`-test.*` flag。只有被已知 flag 消费的参数或 `--help`/`-h`（触发 `ErrHelp`）被排除。未知 flag 的收集需要 `ParseErrorsAllowlist.UnknownFlags = true`；位置参数和 `--`/`-test.*` 始终收集。通过 `f.UnknownFlags()` 在 `Parse()` 后访问。完整设计见 `docs/superpowers/specs/2026-03-12-unknown-flags-collection-design.md`。

### 测试

部分测试依赖全局状态（`CommandLine`）和执行顺序——CI 中禁用了 test shuffle。`export_test.go` 提供 `ResetForTesting` 用于在测试间重置 `CommandLine` 状态。未消费参数的测试在 `unknown_flags_test.go` 中（与 `flag_test.go` 分离）。

## Lint 配置

在 `.golangci.yaml` 中配置（golangci-lint v2）：errorlint、nolintlint、revive、unconvert、unparam、gofmt。使用 `//nolint` 抑制 lint 时需添加理由注释（由 nolintlint 强制）。
