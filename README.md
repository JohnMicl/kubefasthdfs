# kubefasthdfs

# store using mem
# bootstrap seed node
```bash
./kubefasthdfs -id node0 -inmem ~/testkubefasthdfs/node0
```

## using rocksdb to store data
./kubefasthdfs -id node0 ~/testkubefasthdfs/node0

#
```bash
curl -XPOST localhost:11000/key -d '{"user1": "batman"}'
curl -XGET localhost:11000/key/user1
```

add other node
bool值仅需提供标志 -v，无需指定任何值。flag 包会检测到 -v 标志的存在，并将 verbose 变量设为 true。
总之，传入 flag 包中的布尔型参数时，只需在命令行中指定对应的布尔标志（如 -v），无需提供额外的值。标志的存在表示布尔值为 true，否则保持默认值（通常为 false）。
否则会有异常 -inmem true之类的写法

```bash
./kubefasthdfs -id node1 -inmem -haddr localhost:11001 -raddr localhost:12001 -join localhost:11000 ~/testkubefasthdfs/node1
./kubefasthdfs -id node2 -inmem -haddr localhost:11002 -raddr localhost:12002 -join localhost:11000 ~/testkubefasthdfs/node2
```

## using rocksdb to store data
```bash
./kubefasthdfs -id node1 -haddr localhost:11001 -raddr localhost:12001 -join localhost:11000 ~/testkubefasthdfs/node1
./kubefasthdfs -id node2 -haddr localhost:11002 -raddr localhost:12002 -join localhost:11000 ~/testkubefasthdfs/node2
```

# test other node
```bash
curl -XGET localhost:11000/key/user1
curl -XGET localhost:11001/key/user1
curl -XGET localhost:11002/key/user1
```