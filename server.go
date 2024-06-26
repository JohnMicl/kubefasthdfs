package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"kubefasthdfs/config"
	"kubefasthdfs/httpd"
	"kubefasthdfs/logger"
	"kubefasthdfs/store"
	"kubefasthdfs/utils"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
)

var confYaml = flag.String("c", "conf.yaml", "config file name")

// Command line defaults
const (
	DefaultHTTPAddr = "localhost:11000"
	DefaultRaftAddr = "localhost:12000"
)

// Command line parameters
var inmem bool
var httpAddr string
var raftAddr string
var joinAddr string
var nodeID string

func init() {
	flag.BoolVar(&inmem, "inmem", false, "Use in-memory storage for Raft")
	flag.StringVar(&httpAddr, "haddr", DefaultHTTPAddr, "Set the HTTP bind address")
	flag.StringVar(&raftAddr, "raddr", DefaultRaftAddr, "Set Raft bind address")
	flag.StringVar(&joinAddr, "join", "", "Set join address, if any")
	flag.StringVar(&nodeID, "id", "", "Node ID. If not set, same as Raft bind address")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <raft-data-path> \n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
}

func main() {
	err := utils.ParseYamlFile(*confYaml, &config.Config)
	if err != nil {
		fmt.Printf("Parse yaml failed =%+v\n", err)
		return
	}
	// 初始化日志
	err = logger.InitLogger(config.Config.LogInfo)
	if err != nil {
		panic(fmt.Sprintf("logger init err:%v", err))
	}
	logger.Logger.Info("logger init success!")

	logger.Logger.Info(fmt.Sprintf("cmd info only joinAddr=%+v\n", joinAddr))

	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "No Raft storage directory specified\n")
		os.Exit(1)
	}

	logger.Logger.Info(fmt.Sprintf("cmd info, inmme=%+v, httpAddr=%+v, raftAddr=%+v, joinAddr=%+v, nodeId=%+v\n", inmem, httpAddr, raftAddr, joinAddr, nodeID))
	if nodeID == "" {
		nodeID = raftAddr
	}

	// Ensure Raft storage exists.
	raftDir := flag.Arg(0)
	if raftDir == "" {
		log.Fatalln("No Raft storage directory specified")
	}
	if err := os.MkdirAll(raftDir, 0700); err != nil {
		log.Fatalf("failed to create path for Raft storage: %s", err.Error())
	}

	rocksdbDir := filepath.Join(raftDir, "rocksdb")
	s, err := store.NewStore(inmem, rocksdbDir)
	if err != nil {
		log.Fatalf("failed to create path for rocksdb storage: %s, dir= %s", err.Error(), rocksdbDir)
	}

	s.RaftDir = raftDir
	s.RaftBind = raftAddr
	if err := s.Open(joinAddr == "", nodeID); err != nil {
		log.Fatalf("failed to open store: %s, joinAddr: %s, nodeId: %s", err.Error(), joinAddr, nodeID)
	}

	h := httpd.New(httpAddr, s)
	if err := h.Start(); err != nil {
		log.Fatalf("failed to start HTTP service: %s", err.Error())
	}

	// If join was specified, make the join request.
	if joinAddr != "" {
		if err := join(joinAddr, raftAddr, nodeID); err != nil {
			log.Fatalf("failed to join node at %s: %s", joinAddr, err.Error())
		}
	}

	// We're up and running!
	log.Printf("hraftd started successfully, listening on http://%s", httpAddr)

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	log.Println("hraftd exiting")
}

func join(joinAddr, raftAddr, nodeID string) error {
	b, err := json.Marshal(map[string]string{"addr": raftAddr, "id": nodeID})
	if err != nil {
		return err
	}
	resp, err := http.Post(fmt.Sprintf("http://%s/join", joinAddr), "application-type/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
