# 项目简介
mini_kv，使用hashicorp/raft，实现一个简单的分布式KV存储服务
# 目录结构
```text
├── cache.go 内存KV存储
├── cluster.go raft集群构建
├── fsm.go raft log状态机
├── http.go http服务
├── main.go 启动
├── options.go 命令可选项
├── run.sh 
└── snapshot.go 快照管理
```
# 使用
## 初始化

在项目根目录下
```shell script
go build -o minikv ./
```
- 新建leader节点
```shell script
./minikv --http=127.0.0.1:6000 --raft=127.0.0.1:7000 --node=1 --bootstrap=true
```
- 新建第1个follower节点
```shell script
./minikv --http=127.0.0.1:6001 --raft=127.0.0.1:7001 --node=2 --join=127.0.0.1:6000
```
- 新建第2个follower节点
```shell script
./minikv --http=127.0.0.1:6002 --raft=127.0.0.1:7002 --node=3 --join=127.0.0.1:6000
```
## 数据同步
- 向leader节点发出写请求
```shell script
curl http://localhost:6000/set?key=ping&value=pong
curl http://localhost:6000/set?key=ping1&value=pong1
```
- 向任意一个follower节点发出读请求，验证数据同步
```shell script
curl http://localhost:6001/get?key=ping
```
## 快照恢复

终止3个节点（control+C），然后重启三个节点。
重启后，节点会找到对应的文件目录，读取快照，恢复数据；读取节点信息，恢复集群关系
```shell script
./minikv --http=127.0.0.1:6000 --raft=127.0.0.1:7000 --node=1
./minikv --http=127.0.0.1:6001 --raft=127.0.0.1:7001 --node=2
./minikv --http=127.0.0.1:6002 --raft=127.0.0.1:7002 --node=3
```
## leader切换
终止leader节点，两个follower节点会开始选举，其中一个会成为leader，提供写服务。
leader节点会一直尝试连接掉线的旧leader节点。
## 故障恢复

### leader故障恢复
重启旧的leader节点
```shell script
./minikv --http=127.0.0.1:6000 --raft=127.0.0.1:7000 --node=1
```
旧leader节点重启后，首先进行leader选举状态，然后发现自己的选举任期比现存的选举任期小，
于是放弃选举，进入follower状态，在新leader的心跳到来之前，旧leader因为超时进入candidate状态，然后一直发起选举，
其他节点因为已经有leader，所以会一直拒绝给旧leader投票。
终于旧leader收到了新leader的心跳，退出candidate，进入follower状态。
### follwer故障恢复
leader节点不会变，follower节点恢复后会进入follower状态，不会尝试leader选举。但是因为心跳超时进入candidate，
经过几次失败的leader选举后，收到了leader的心跳，然后重新进入follower状态。