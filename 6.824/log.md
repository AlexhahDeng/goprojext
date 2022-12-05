# 实验过程

```
go build -race -buildmode=plugin ../mrapps/wc.go
rm mr-out* //删除所有输出文件

go run -race mrcoordinator.go pg-*.txt

// 在其他的几个窗口运行
go run -race mrworker.go wc.so

// 查看结果
cat mr-out-* | sort | more
```

# 一些要求
coordinator和worker的主routine在`main/mrcoordinator.go` and `main/mrworker.go`，别动

只修改`mr/coordinator.go`, `mr/worker.go`, `mr/rpc.go`
可以借用`wc.go`的内容

# a few rules

- map过程要把中间key分为nReduce块（因为有nReduce个reduce任务，其中nReduce是由main/mrcoordinator.go传给MakeCoordinator()的
- reduce worker的实现应该把第X个reduce任务的输出放到mr-out-X中
- 一个mr-out-X文件应该包含
- map worker 应该把输出的文件放在同一目录，方便reduce来查收
- main/mrcoordinator.go期望mr/coordinator.go实现一个Done()方法，当master工作结束的时候，返回true



# hints

1. 首先修改mr/worker.go的worker()发送rpc请求给coordinator请求任务，然后修改coordinator做出回应（返回还没分配出去的文件名），然后修改worker来读取文件内容并做出处理
2. 如果修改了mr下的任何东西，都需要重新去build插件哦
3. 中间文件最好命名为mr-X-Y，其中X是map任务编号，Y是reduce任务编号
4. map worker可以用ihash(key)函数来选择对应key的reduce worker
5. coordinator是作为rpc服务器，是并发的，注意共享数据加锁
6. workers可能会需要等待，因为reduce要等到所有的map都干完活才能开始。一个简单的方法是，workers周期性询问coordinator要活儿干，用time.Sleep()来划分间隙。还有个办法以后再说
7. coordinator没法区分崩掉的worker和因为某些原因而停下的worker，以及干活干的很慢的worker，因此最好的办法就是让coordinator等待固定时间，如果worker的活儿没干完，就重新分配给别人，对于这个实验，让coordinator等待10s，然后就假定worker挂了。
8. ❓为了确保没有人在crashes情况下读到只写了一部分的文件，可以采取mapreduce论文的策略，用ioutil.TempFile来创建临时文件，然后在写完的时候用os.Rename来改名
9. 

# 一些想法

* 判断map worker的活儿是否干完
    1. 过了10s看文件是否存在
    2. 直接查看coordinator中maplist对应文件的状态

* 修改文件状态的时候可能要考虑加锁

```
\\ go判断文件是否存在
if _, err := os.Stat(logFile); os.IsNotExist(err) {
    return
}
```

worker会一直去问coordinator是否有活儿干，如果没有，就等着
如果有，coordinator根据情况给他们分配，map or reduce的工作


分发任务阶段怎么加锁比较高效是个问题
粒度大了，没法并发
粒度小了，开销太大