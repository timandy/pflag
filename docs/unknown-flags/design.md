# 设计文档

## 整体思路

在 `FlagSet` 结构体中新增一个 `unknownFlags []string` 字段。解析过程中，在每个"参数未被已知 flag 消费"的代码路径上，把该参数追加到这个 slice 里。解析结束后通过 `UnknownFlags()` 方法访问。

核心设计决策：**不修改任何现有逻辑，只在现有分支中增加记录操作。**

## API

```go
// FlagSet 方法——返回内部 slice 的直接引用（与 Args() 风格一致）
func (f *FlagSet) UnknownFlags() []string

// 顶层函数——委托给 CommandLine
func UnknownFlags() []string
```

## 实现细节

### 初始化

每次 `Parse()` / `ParseAll()` 调用时重置：

```go
f.unknownFlags = make([]string, 0, len(arguments))
```

这确保多次调用 `Parse()` 不会累积旧数据。

### `stripUnknownFlagValue` 方法

原本是一个独立函数，只负责 strip 未知 flag 后面的值参数。重构为 `FlagSet` 方法，同时负责记录和 strip：

```go
func (f *FlagSet) stripUnknownFlagValue(flag string, args []string) []string {
f.unknownFlags = append(f.unknownFlags, flag)

if len(args) == 0 {
return args
}
first := args[0]
if len(first) > 0 && first[0] == '-' {
return args
}

f.unknownFlags = append(f.unknownFlags, first)
if len(args) > 1 {
return args[1:]
}
return nil
}
```

调用方只需一行：

```go
// parseLongArg 中
return f.stripUnknownFlagValue("--"+name, a), nil

// parseSingleShortArg 中
outArgs = f.stripUnknownFlagValue("-"+string(c), outArgs)
```

### 收集点一览

解析流程为 `Parse()` → `parseArgs()` → `parseLongArg()` / `parseShortArg()` → `parseSingleShortArg()`。

下面按代码路径列出所有收集点：

#### `parseArgs` —— 位置参数

```go
if len(s) == 0 || s[0] != '-' || len(s) == 1 {
f.args = append(f.args, s)
f.unknownFlags = append(f.unknownFlags, s) // ← 新增
continue
}
```

`interspersed = false` 时，首个位置参数和后续所有参数也同样处理。

#### `parseArgs` —— `--` 终止符

```go
if len(s) == 2 { // "--"
f.argsLenAtDash = len(f.args)
f.args = append(f.args, args...)
f.unknownFlags = append(f.unknownFlags, s) // ← 记录 "--"
f.unknownFlags = append(f.unknownFlags, args...) // ← 记录后续参数
break
}
```

#### `parseLongArg` —— 未知长 flag

两条路径：

1. **带 `=`**（如 `--foo=bar`）：直接记录原始字符串
2. **不带 `=`**（如 `--foo bar`）：委托 `stripUnknownFlagValue`

都在 `ParseErrorsAllowlist.UnknownFlags` 的 case 分支中。

#### `parseSingleShortArg` —— 未知短 flag

两条路径：

1. **带 `=`**（如 `-f=arg`）：直接记录 `"-f=arg"`
2. **不带 `=`**（如 `-f`）：委托 `stripUnknownFlagValue`

#### `parseSingleShortArg` —— `-test.*` flag

```go
if isGotestShorthandFlag(shorthands) {
f.unknownFlags = append(f.unknownFlags, "-"+shorthands) // ← 新增
return
}
```

#### `--help` / `-h` —— 不收集

这两个触发 `f.usage()` 并返回 `ErrHelp`，属于被解析器消费的参数，不记录。

### 组合短 flag 的处理

对于 `-xvf`（其中 `-v` 已知，`-x` 和 `-f` 未知）：

- `parseShortArg` 逐字符迭代调用 `parseSingleShortArg`
- `-x` 未知 → 记录 `"-x"`
- `-v` 已知 → 正常消费
- `-f` 未知 → 记录 `"-f"`

最终 `unknownFlags = ["-x", "-f"]`。

## 行为矩阵

标记 `(allowlist)` 的行需要 `ParseErrorsAllowlist.UnknownFlags = true`。

| 输入                       | 已知        | `unknownFlags`           | `args`             | 说明                    |
|--------------------------|-----------|--------------------------|--------------------|-----------------------|
| `--known`                | `--known` | `[]`                     | `[]`               | 被已知 flag 消费           |
| `--foo=bar`              | 无         | `["--foo=bar"]`          | `[]`               | (allowlist)           |
| `--foo bar`              | 无         | `["--foo", "bar"]`       | `[]`               | (allowlist)           |
| `--foo --next`           | `--next`  | `["--foo"]`              | `[]`               | (allowlist)           |
| `--foo`（末尾）              | 无         | `["--foo"]`              | `[]`               | (allowlist)           |
| `--foo=bar pos`          | 无         | `["--foo=bar", "pos"]`   | `["pos"]`          | (allowlist) pos 是位置参数 |
| `-f=arg`                 | 无         | `["-f=arg"]`             | `[]`               | (allowlist)           |
| `-f`（末尾）                 | 无         | `["-f"]`                 | `[]`               | (allowlist)           |
| `-f bar`                 | 无         | `["-f", "bar"]`          | `[]`               | (allowlist)           |
| `-rf`                    | `-r`      | `["-f"]`                 | `[]`               | (allowlist) 部分组合      |
| `-xvf`                   | `-v`      | `["-x", "-f"]`           | `[]`               | (allowlist)           |
| `--help`                 | 无         | `[]`                     | `[]`               | 已消费：usage + ErrHelp   |
| `-h`                     | 无         | `[]`                     | `[]`               | 已消费：usage + ErrHelp   |
| `--`                     | —         | `["--"]`                 | `[]`               | 终止符                   |
| `-- --foo pos`           | —         | `["--", "--foo", "pos"]` | `["--foo", "pos"]` | 终止符后                  |
| `-test.v`                | —         | `["-test.v"]`            | `[]`               | go test flag          |
| `arg`                    | —         | `["arg"]`                | `["arg"]`          | 位置参数                  |
| `a b c`                  | —         | `["a", "b", "c"]`        | `["a", "b", "c"]`  | 多个位置参数                |
| `--foo`（allowlist=false） | 无         | `[]`                     | `[]`               | 返回错误                  |

## 变更文件

| 文件                      | 变更内容                                       |
|-------------------------|--------------------------------------------|
| `flag.go`               | 新增字段、访问器、重构 `stripUnknownFlagValue`、修改解析函数 |
| `unknown_flags_test.go` | 独立测试文件，覆盖所有场景                              |
| `go.mod` / `go.sum`     | 新增 `testify` 依赖（仅测试）                       |
