# authentication required 修复说明

v0.1.3 修复“创建任何内容都显示 authentication required”的问题。

常见原因：

- 浏览器里保存的是旧 session，主控重启或数据文件变化后 session 已失效。
- 浏览器缓存了旧版 `web/app.js`。
- 经过 Nginx、宝塔、CDN 或其他反代后，`Authorization: Bearer <token>` 请求头没有被传递到主控。

本版本增加了兼容认证方式：

- `Authorization: Bearer <session>`
- `X-YFCDN-Session: <session>`
- `X-Auth-Token: <session>`
- `yfcdn_session` HttpOnly Cookie
- `access_token` 查询参数

Web 控制台会同时发送 `Authorization` 和 `X-YFCDN-Session`，登录成功时后端也会写入 `yfcdn_session` Cookie。即使某些代理丢弃 `Authorization` 头，创建操作也能正常认证。

升级后执行：

```bash
cd /root/YFCDN
pkill -f 'yfcdn-control|go run ./cmd/control' || true
GOTOOLCHAIN=local go run ./cmd/control -addr :8080 -data data/yfcdn.json -static web
```

然后浏览器按 `Ctrl + F5` 强制刷新，重新登录后再创建节点/域名。
