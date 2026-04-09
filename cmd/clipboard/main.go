package main

import (
	"clipboard/internal/config"
	"clipboard/internal/rpc"
	"clipboard/internal/service"
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"golang.org/x/crypto/argon2"
)

func NewKey(password string) []byte {
	return argon2.IDKey(
		[]byte(password),
		[]byte("clipboard-sharing-service"),
		3,
		64*1024,
		4,
		32,
	)
}

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		panic(err)
	}
	key := NewKey(cfg.Password)
	localService, err := service.NewLocal(cfg.Peers, key)
	if err != nil {
		panic(err)
	}
	rpcServer, err := rpc.NewServer(cfg.Bind, key)
	if err != nil {
		panic(err)
	}
	err = rpcServer.Register(service.NewRPC(localService))
	if err != nil {
		panic(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	ctx, stop := context.WithCancel(context.Background())
	go func() {
		localService.Start(ctx)
		wg.Done()
	}()
	<-localService.Ready()
	go func() {
		err := rpcServer.ListenAndServe()
		if err != nil {
			panic(err)
		}
		wg.Done()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	stop()
	localService.Stop()
	_ = rpcServer.Close()

	wg.Wait()
}
