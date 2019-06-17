# terminal-encrypt-chat

一个终端中的端到端加密聊天工具

- 使用 ECDH 密钥协商算法生成对称加密密钥（由 [curve25519](https://godoc.org/golang.org/x/crypto/curve25519) 库实现)
- 使用对称加密算法加密聊天内容(由 [chacha20poly1305](https://godoc.org/golang.org/x/crypto/chacha20poly1305)库实现)

# Usage

### Server

默认监听在 `0.0.0.0:9468`, 可以使用 `-h` 参数指定:
```bash
./server -h=ip:port
```

### client
必须指定 ID 和 服务器地址，双方 ID 一致即可建立连接
```bash
./client -i=ID -h=ip:port
```

# Download

去 [Releases](https://github.com/guyanyijiu/terminal-encrypt-chat/releases) 页面下载编译好的可执行文件