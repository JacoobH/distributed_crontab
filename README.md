# distributed_crontab


![](https://img.shields.io/badge/update-today-blue.svg) ![](https://img.shields.io/badge/gitbook-making-lightgrey.svg)
<div align="center">
    <a href="https://github.com/JacoobH/distributed_crontab"> <img src="https://badgen.net/github/stars/JacoobH/distributed_crontab?icon=github&color=4ab8a1"></a>
    <a href="https://github.com/JacoobH/distributed_crontab"> <img src="https://badgen.net/github/forks/JacoobH/distributed_crontab?icon=github&color=4ab8a1"></a>
    
</div>


| 系统介绍 |项目展示|技术栈|环境参数
| :---: | :----: | :----: | :----: |
| [:computer:](#computer-系统介绍)  | [:bulb:](#bulb-项目展示)|[:memo:](#memo-技术栈)|[:wrench:](#wrench-环境参数)|

## :computer: 系统介绍

### 传统crontab痛点及本项目的解决方案

在介绍本系统之前，先分析一下在使用 传统crontab 的时候会遇到的几类问题
1. 配置任务时，需要ssh登录脚本服务器来进行操作
2. 服务器宕机，任务终将中止调度，需要人工迁移
3. 排查问题低效，无法方便查看任务状态与错误输出

本系统解决了上面的三个问题：
1. 进行web可视化，方便管理
2. 分布式架构、集群化调度、不存在单点故障
3. 追踪任务执行状态，采集任务输出，可视化log查看

### 系统架构
<div align="center"> <img src="https://github.com/JacoobH/images/blob/main/images/distributed_crontab/%E6%9E%B6%E6%9E%84.png"/> </div><br>

## :bulb: 项目展示 

正在部署....

## :memo: 技术栈 

### 后台

- 开发语言：golang
- mongodb进行日志存储及管理
- etcd实现任务同步、服务发现及调度互斥
- 使用[Golang Cron expression parser](https://github.com/gorhill/cronexpr)解析cron表达式
- web框架[Gin](https://github.com/gin-gonic/gin)

### 前端

- jqury
- bootstarp

## :wrench: 环境参数

- go version go1.18.1 
- etcd-v3.3.8
- MONGO_VERSION=5.0.5
- 开发环境: Goland, Ubuntu 20.04.4 LTS
- 部署环境: Centos7.9
