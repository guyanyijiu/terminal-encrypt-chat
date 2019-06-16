# terminal-encrypt-chat
一个终端中的端到端加密聊天工具

- 使用 ECDH 密钥协商算法生成对称加密密钥（由 [curve25519](https://godoc.org/golang.org/x/crypto/curve25519) 库实现)
- 使用对称加密算法加密聊天内容(由 [chacha20poly1305](https://godoc.org/golang.org/x/crypto/chacha20poly1305)库实现)