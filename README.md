# YFCDN

YFCDN 是一个轻量级 CDN 控制面板原型，目标是整合轻量节点管理、多租户运营、运营商/API 调用、域名配置下发和边缘 Nginx 缓存能力。

当前版本：`v0.1.3`

## v0.1.3 修复内容

- 默认语言为中文简体，支持简体中文 / English 自行切换。
- 修复前端菜单和按钮点击体验，重渲染后按钮不会失效。
- 修复创建租户、运营商、节点、域名、套餐、API Key 时可能出现的 `authentication required`。
- Web 控制台同时发送 `Authorization` 和 `X-YFCDN-Session`。
- 后端登录时写入 `yfcdn_session` HttpOnly Cookie。
- 后端支持 `Authorization`、`X-YFCDN-Session`、`X-Auth-Token`、Cookie、`access_token` 多种认证来源。
- 节点安装页面提供一键安装命令。
- 支持远程边缘节点安装。
- 支持主控和被控节点部署在同一台服务器。
- 控制面板通过 `/install-node.sh` 提供节点安装脚本。

## 启动主控

```bash
go run ./cmd/control -addr :8080 -data data/yfcdn.json -static web
```

默认账号：

```text
admin
admin123
```

建议公网部署时设置强密码：

```bash
YFCDN_ADMIN_PASSWORD='你的强密码' go run ./cmd/control -addr :8080 -data data/yfcdn.json -static web
```

访问：

```text
http://服务器IP:8080
```

## 升级说明

保留现有数据文件，只覆盖源码：

```bash
cd /root
cp -a YFCDN YFCDN.bak.$(date +%F-%H%M)
unzip -o YFCDN-v0.1.3.zip
rsync -a --exclude 'data/yfcdn.json' YFCDN-v0.1.3/ YFCDN/
cd /root/YFCDN
pkill -f 'yfcdn-control|go run ./cmd/control' || true
GOTOOLCHAIN=local go run ./cmd/control -addr :8080 -data data/yfcdn.json -static web
```

升级后浏览器按 `Ctrl + F5` 强制刷新，并重新登录。

## 一键安装节点

先在控制台进入“节点管理”，创建一个节点。系统会生成 `node_key`。

### 远程边缘节点

在目标边缘服务器执行控制台给出的命令，格式如下：

```bash
curl -fsSL http://主控IP:8080/install-node.sh | sudo bash -s -- --control http://主控IP:8080 --node-key node_xxx
```

脚本会自动完成：

- 安装 Nginx、curl、Go 编译器
- 编译 `yfcdn-agent`
- 写入 `/usr/local/bin/yfcdn-agent`
- 写入 `/etc/yfcdn/agent.env`
- 写入 systemd 服务 `yfcdn-agent`
- 启动 Nginx 和 Agent
- Agent 自动上报心跳并拉取 Nginx 配置

### 主控和节点在同一台服务器

在主控所在服务器执行：

```bash
curl -fsSL http://127.0.0.1:8080/install-node.sh | sudo bash -s -- --control http://127.0.0.1:8080 --node-key node_xxx --same-server
```

同机部署说明：

- 主控默认监听 `8080`
- Nginx 默认监听 `80`
- Agent 使用 `127.0.0.1:8080` 连接主控
- 主控和节点不会抢占同一个端口

## 手动构建

```bash
go build -o bin/yfcdn-control ./cmd/control
go build -o bin/yfcdn-agent ./cmd/agent
```

## Docker

```bash
docker compose up --build
```

## 常见问题

### 创建任何内容都提示 authentication required

升级到 v0.1.3，重启主控，然后浏览器按 `Ctrl + F5` 强制刷新并重新登录。

常见原因是旧 token、浏览器缓存旧版前端，或反向代理没有传递 `Authorization` 请求头。v0.1.3 已加入双通道认证：前端同时发送 `Authorization` 和 `X-YFCDN-Session`，后端也会写入 `yfcdn_session` HttpOnly Cookie。

### 左侧功能点击无反应

请确认浏览器加载的是新版本前端。升级后建议强制刷新：

```text
Ctrl + F5
```

### 节点一直离线

检查：

```bash
systemctl status yfcdn-agent --no-pager
journalctl -u yfcdn-agent -f
curl http://主控IP:8080/api/v1/health
```

### Nginx 配置没有生效

检查：

```bash
nginx -t
cat /etc/nginx/conf.d/yfcdn-managed.conf
systemctl reload nginx
```

## 同步源码到 GitHub

如果仓库是刚初始化的空项目或只有一个默认 README，可以在项目根目录执行：

```bash
FORCE=1 ./scripts/sync-github.sh https://github.com/ww485000/YFCDN.git main
```

如果使用 SSH：

```bash
FORCE=1 ./scripts/sync-github.sh git@github.com:ww485000/YFCDN.git main
```

如果仓库已有重要提交，不要使用 `FORCE=1`，请先 `git pull` 合并后再推送。

## 后续方向

- 数据库替换 JSON 存储
- RBAC 管理员/运营商/租户权限
- 域名 TXT 验证
- ACME 自动证书
- 流量统计、账单和套餐限额
- 缓存刷新 API
- WAF/CC 防护规则
- 智能 DNS 和节点调度
