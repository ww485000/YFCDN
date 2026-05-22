# YFCDN 节点安装

YFCDN v0.1.3 开始提供一键节点安装脚本。

## 远程边缘节点

```bash
curl -fsSL http://主控IP:8080/install-node.sh | sudo bash -s -- --control http://主控IP:8080 --node-key node_xxx
```

## 主控和节点同机

```bash
curl -fsSL http://127.0.0.1:8080/install-node.sh | sudo bash -s -- --control http://127.0.0.1:8080 --node-key node_xxx --same-server
```

## 脚本参数

```text
--control URL        主控地址
--node-key KEY      节点密钥
--same-server       主控和节点在同一台服务器
--nginx-conf PATH   Nginx 配置输出路径
--reload-cmd CMD    Nginx 重载命令
```

## 安装后检查

```bash
systemctl status yfcdn-agent --no-pager
journalctl -u yfcdn-agent -f
nginx -t
cat /etc/nginx/conf.d/yfcdn-managed.conf
```
