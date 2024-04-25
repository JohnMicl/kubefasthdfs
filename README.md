# kubefasthdfs

# store using mem
# bootstrap seed node
```bash
./kubefasthdfs -id node0 -inmem true ~/testkubefasthdfs/node0
```

#
```bash
curl -XPOST localhost:11000/key -d '{"user1": "batman"}'
curl -XGET localhost:11000/key/user1
```

# add other node
```bash
./kubefasthdfs -id node1 -inmem true -haddr localhost:11001 -raddr localhost:12001 -join :11000 ~/testkubefasthdfs/node1
./kubefasthdfs -id node2 -inmem true -haddr localhost:11002 -raddr localhost:12002 -join :11000 ~/testkubefasthdfs/node2
```

# test other node
```bash
curl -XGET localhost:11000/key/user1
curl -XGET localhost:11001/key/user1
curl -XGET localhost:11002/key/user1
```