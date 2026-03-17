# 需求说明

## 背景

`pflag` 当前的解析行为：

- **已知 flag**（如 `--verbose`）：被解析消费，值存入对应变量
- **位置参数**（如 `file.txt`）：存入 `Args()`
- **未知 flag**（如 `--debug`）：
    - `ParseErrorsAllowlist.UnknownFlags = false`（默认）：报错
    - `ParseErrorsAllowlist.UnknownFlags = true`：静默丢弃

问题在于第三种情况。当设置 allowlist 允许未知 flag 时，这些 flag 被静默丢弃了，调用方完全不知道用户传了什么未知参数。

## 需求

在 `FlagSet` 上新增一个 `UnknownFlags()` 方法，解析完成后返回所有**未被已知 flag 消费**的参数。

### 什么算"未消费"？

| 参数类型                                    | 是否收集 | 说明                           |
|-----------------------------------------|------|------------------------------|
| 未知 flag（如 `--debug`）                    | 是    | 没有匹配到任何已定义的 flag             |
| 未知 flag 的值（如 `--debug val` 中的 `val`）    | 是    | 被 strip 作为未知 flag 的值         |
| 位置参数（如 `file.txt`）                      | 是    | 不属于任何 flag                   |
| `--` 终止符                                | 是    | 本身未被任何 flag 消费               |
| `--` 后的所有参数                             | 是    | 终止符后不再解析 flag                |
| `-test.v` 等 Go test flag                | 是    | 被静默跳过，未被消费                   |
| 已知 flag（如 `--verbose`）                  | 否    | 被已知 flag 定义消费                |
| 已知 flag 的值（如 `--output file` 中的 `file`） | 否    | 被已知 flag 消费作为值               |
| `--help` / `-h`（未定义时）                   | 否    | 触发 usage 输出和 `ErrHelp`，属于已消费 |

### 关键约束

1. **不改变任何现有行为**。`Args()`、错误处理、`--help`/`-h`、`--` 终止符等行为完全不变
2. **`unknownFlags` 是纯增量的额外记录**，与 `args` 可以重叠（位置参数同时出现在两者中）
3. **API 风格与 `Args()` 保持一致**：`f.UnknownFlags()` 返回 slice 的直接引用，只在 `Parse()` 后调用

### 与 allowlist 的关系

- 未知 flag 的收集依赖 `ParseErrorsAllowlist.UnknownFlags = true`（否则未知 flag 直接报错，不存在"收集"的场景）
- 位置参数、`--` 终止符、`-test.*` 的收集不依赖 allowlist 设置

## 边界情况

### 未知 flag 值的判定

未知 flag 后面的参数是否是它的"值"，取决于该参数是否以 `-` 开头：

| 输入             | `unknownFlags`     | 说明                                      |
|----------------|--------------------|-----------------------------------------|
| `--foo bar`    | `["--foo", "bar"]` | `bar` 不以 `-` 开头 → 被 strip 为 `--foo` 的值  |
| `--foo --next` | `["--foo"]`        | `--next` 以 `-` 开头 → 不是值，作为下一个 flag 继续解析 |
| `--foo`（末尾）    | `["--foo"]`        | 后面没有参数 → 只记录 flag 本身                    |
| `--foo=bar`    | `["--foo=bar"]`    | `=` 形式 → 值嵌入在 flag 字符串中，整体记录            |

**重要**：这个判定逻辑是 `stripUnknownFlagValue` 的现有行为，不是新逻辑。`unknownFlags` 只是记录其结果。

### `=` 形式 vs 空格分隔形式

| 输入              | `unknownFlags`         | `args`    | 说明                                |
|-----------------|------------------------|-----------|-----------------------------------|
| `--foo=bar pos` | `["--foo=bar", "pos"]` | `["pos"]` | `=` 形式值已嵌入，`pos` 是独立位置参数          |
| `--foo bar pos` | `["--foo", "bar"]`     | `["pos"]` | 空格形式，`bar` 被 strip 为值，`pos` 是位置参数 |
| `-f=arg pos`    | `["-f=arg", "pos"]`    | `["pos"]` | 短 flag `=` 形式                     |
| `-f bar pos`    | `["-f", "bar"]`        | `["pos"]` | 短 flag 空格形式                       |

### 组合短 flag

短 flag 可以组合使用（如 `-rf`）。解析器逐字符拆分处理：

| 输入        | 已知         | `unknownFlags`  | 说明                         |
|-----------|------------|-----------------|----------------------------|
| `-rf`     | `-r`（bool） | `["-f"]`        | `-r` 被消费，`-f` 未知           |
| `-fag`    | `-a`（bool） | `["-f", "-g"]`  | `-a` 被消费，`-f` 和 `-g` 未知    |
| `-f`      | 无          | `["-f"]`        | 单个未知短 flag                 |
| `-rf val` | `-r`（bool） | `["-f", "val"]` | `-f` 未知且 `val` 被 strip 为其值 |

### `--` 终止符

`--` 是 POSIX 标准的 flag 终止符。遇到后停止 flag 解析，后续所有参数进入 `args`。

| 输入                 | `unknownFlags`           | `args`             | 说明               |
|--------------------|--------------------------|--------------------|------------------|
| `--`               | `["--"]`                 | `[]`               | 仅终止符，无后续参数       |
| `-- --foo pos`     | `["--", "--foo", "pos"]` | `["--foo", "pos"]` | 终止符 + 后续全部收集     |
| `-- -rf pos`       | `["--", "-rf", "pos"]`   | `["-rf", "pos"]`   | 终止符后不拆分短 flag 组合 |
| `--known -- --foo` | `["--", "--foo"]`        | `["--foo"]`        | 已知 flag 在终止符前被消费 |

### `--help` / `-h`

当 `--help` 或 `-h` **未被用户定义为 flag** 时，解析器内部消费它们（触发 usage + `ErrHelp`）。

| 输入                             | `unknownFlags` | 错误        | 说明                        |
|--------------------------------|----------------|-----------|---------------------------|
| `--help`                       | `[]`           | `ErrHelp` | 被消费，不收集                   |
| `-h`                           | `[]`           | `ErrHelp` | 被消费，不收集                   |
| `--foo --help`（allowlist=true） | `["--foo"]`    | `ErrHelp` | `--foo` 被收集，`--help` 终止解析 |

如果用户**自己定义了** `--help` 或 `-h` flag，则它们作为已知 flag 正常消费，不涉及此逻辑。

### `-test.*` Go test flag

Go test 框架传入的 flag（如 `-test.v`、`-test.run=xxx`）被 `isGotestShorthandFlag` 静默跳过。

| 输入                  | `unknownFlags`          | 说明           |
|---------------------|-------------------------|--------------|
| `-test.v`           | `["-test.v"]`           | 被静默跳过但未消费，收集 |
| `-test.run=TestFoo` | `["-test.run=TestFoo"]` | 同上，整个字符串收集   |

### `interspersed` 模式

`interspersed`（默认 `true`）控制位置参数和 flag 能否交错出现。

**`interspersed = true`（默认）**：位置参数后可以继续解析 flag。

| 输入                                 | 已知         | `unknownFlags`                  | `args`             | 说明                                           |
|------------------------------------|------------|---------------------------------|--------------------|----------------------------------------------|
| `--known pos1 --unknown pos2`      | `--known`  | `["pos1", "--unknown", "pos2"]` | `["pos1"]`         | `pos1` 是位置参数，`pos2` 被 strip 为 `--unknown` 的值 |
| `a b c`                            | 无          | `["a", "b", "c"]`               | `["a", "b", "c"]`  | 全部是位置参数                                      |
| `pos1 --foo=bar --output out pos2` | `--output` | `["pos1", "--foo=bar", "pos2"]` | `["pos1", "pos2"]` | 已知 flag 消费，其余收集                              |

**`interspersed = false`**：遇到第一个位置参数后停止 flag 解析，剩余全部进入 `args`。

| 输入                      | 已知        | `unknownFlags`         | `args`                 | 说明               |
|-------------------------|-----------|------------------------|------------------------|------------------|
| `--known arg --unknown` | `--known` | `["arg", "--unknown"]` | `["arg", "--unknown"]` | `arg` 触发停止，后续不解析 |
| `arg --known`           | 无         | `["arg", "--known"]`   | `["arg", "--known"]`   | 首个就是位置参数，全部停止    |

### `NormalizeFunc`

如果设置了 `NormalizeFunc`，flag 名称会先经过标准化再查找。标准化后匹配到已知 flag 的不会被收集。

| 输入             | NormalizeFunc | 已知 flag   | `unknownFlags`     | 说明              |
|----------------|---------------|-----------|--------------------|-----------------|
| `--my_flag`    | `_` → `-`     | `my-flag` | `[]`               | 标准化后匹配，不收集      |
| `--MyFlag`     | `ToLower`     | `myflag`  | `[]`               | 标准化后匹配，不收集      |
| `--other_flag` | `_` → `-`     | 无         | `["--other_flag"]` | 标准化后仍未匹配，收集原始形式 |

### `ParseErrorsWhitelist` 兼容

废弃的 `ParseErrorsWhitelist.UnknownFlags` 与 `ParseErrorsAllowlist.UnknownFlags` 效果相同（代码中通过 `fallthrough` 处理）。两者任一为 `true` 即可触发未知 flag 收集。

### `Parse()` 重复调用

每次调用 `Parse()` 或 `ParseAll()` 都会重置 `unknownFlags`，不会累积上次解析的结果。

| 操作                     | `unknownFlags` |
|------------------------|----------------|
| 第一次 `Parse(["--foo"])` | `["--foo"]`    |
| 第二次 `Parse([])`        | `[]`           |

### `ParseAll` 回调模式

`ParseAll` 使用回调函数处理每个已知 flag。未知 flag 的收集行为与 `Parse` 完全一致。

### allowlist = false 时

未知 flag 直接返回错误，不会被收集。`unknownFlags` 保持为空。但位置参数仍然会在报错前被收集（如果位置参数出现在未知 flag 之前）。

| 输入          | `unknownFlags` | 错误           | 说明            |
|-------------|----------------|--------------|---------------|
| `--foo`     | `[]`           | 返回未知 flag 错误 | 直接报错，不收集      |
| `pos --foo` | `["pos"]`      | 返回未知 flag 错误 | `pos` 在错误前被处理 |

### 空参数和单字符 `-`

| 输入   | `unknownFlags` | `args`  | 说明                         |
|------|----------------|---------|----------------------------|
| `""` | `[""]`         | `[""]`  | 空字符串被当作位置参数                |
| `-`  | `["-"]`        | `["-"]` | 单字符 `-` 被当作位置参数（常表示 stdin） |

### 顶层函数

`UnknownFlags()` 顶层函数委托给 `CommandLine`（全局默认 `FlagSet`），行为与 `Args()` 一致。
