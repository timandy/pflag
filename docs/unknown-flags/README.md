# UnknownFlags —— 未消费参数收集

## 这是什么？

`pflag` 在解析命令行参数时，会把已定义的 flag 消费掉，把位置参数放到 `Args()` 里。但调用方经常需要知道：**哪些参数没有被任何已知 flag 消费？**

`UnknownFlags()` 就是为此而生的。它在解析完成后，返回所有未被已知 flag 消费的参数列表。

## 快速上手

```go
fs := pflag.NewFlagSet("example", pflag.ContinueOnError)
fs.ParseErrorsAllowlist.UnknownFlags = true
fs.Bool("verbose", false, "启用详细输出")

fs.Parse(os.Args[1:])

fmt.Println("已知 flag 处理完毕")
fmt.Println("位置参数:", fs.Args())
fmt.Println("未消费参数:", fs.UnknownFlags())
```

假设命令行是 `myapp --verbose --debug pos1 pos2`：

- `--verbose` 被已知 flag 消费，不出现在 `UnknownFlags()` 中
- `--debug` 是未知 flag，出现在 `UnknownFlags()` 中
- `pos1`、`pos2` 是位置参数，同时出现在 `Args()` 和 `UnknownFlags()` 中

## 使用场景

1. **多级命令转发**：父命令解析自己认识的 flag，把 `UnknownFlags()` 传给子命令或子进程
2. **插件系统**：宿主程序不知道插件会接受哪些 flag，解析后把未消费的转交给插件
3. **调试和日志**：记录用户传了哪些参数没被处理，方便排查问题
4. **渐进式迁移**：老系统迁移过程中，新代码只处理部分 flag，其余的记录下来兼容处理

## 详细文档

- [需求说明](requirements.md) —— 为什么需要这个功能、解决什么问题
- [设计文档](design.md) —— 技术实现细节、收集规则、行为矩阵
