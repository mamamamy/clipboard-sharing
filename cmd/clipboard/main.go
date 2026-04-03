package main

import (
	"bytes"
	"clipboard/internal/clipboard"
	"clipboard/internal/config"
	"clipboard/internal/rpc"
	"clipboard/internal/service"
	"context"
	"log"
	"os"

	"golang.org/x/crypto/argon2"
)

func main() {
	var err error

	log.SetOutput(os.Stdout)

	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("❌ failed to load config: %v", err)
	}

	key := argon2.IDKey([]byte(cfg.Password), []byte("clipboard-sharing"), 3, 64*1024, 4, 32)

	rpcServer := rpc.NewServer(cfg.Bind, key)
	err = rpcServer.Register(&service.RPCClipboardService{})
	if err != nil {
		log.Fatalf("❌ failed to register RPC service: %v", err)
	}

	go func() {
		log.Printf("🚀 RPC server starting, listen on: %s", rpcServer.Addr())
		err := rpcServer.ListenAndServe()
		if err != nil {
			log.Fatalf("❌ RPC server stopped with error: %v", err)
		}
	}()

	rpcClient := rpc.NewClient(cfg.Peer, key)
	go func() {
		log.Printf("🚀 start watching remote clipboard change peer: %s", cfg.Peer)
		for {
			digest, err := service.RemoteClipboardService.WaitLatestDigest(rpcClient)
			if err != nil {
				log.Printf("❌ failed to wait remote latest digest: %v", err)
				continue
			}

			log.Printf("📡 received remote clipboard digest: %s", digest)

			go func() {
				localDigest := service.LocalClipboardService.GetDigest()
				if bytes.Equal(digest, localDigest) {
					log.Println("ℹ️ remote clipboard is consistent with local, no sync required")
					return
				}

				log.Println("🔄 remote clipboard changed, starting sync...")
				info, err := service.RemoteClipboardService.CompareAndGetInfo(rpcClient, localDigest)
				if err != nil {
					log.Printf("❌ failed to get remote clipboard info: %v", err)
					return
				}

				if info.Kind == clipboard.KindNone {
					log.Println("ℹ️ remote clipboard is consistent with local, no sync required")
					return
				} else if info.Kind != clipboard.KindNone {
					clipboard.Write(info)
					log.Printf(
						"✅ successfully synced remote clipboard to local, kind: %s, size: %d, digest: %s",
						info.Kind,
						len(info.Data),
						info.Digest,
					)
				}
			}()
		}
	}()

	watchChan := clipboard.Watch(context.Background())
	log.Println("🚀 start watching clipboard...")

	for {
		select {
		case info, ok := <-watchChan:
			if !ok {
				log.Println("🛑 stop watching clipboard: channel closed")
				return
			}
			log.Printf(
				"📋 local clipboard changed, kind: %s, size: %d, digest: %s",
				info.Kind,
				len(info.Data),
				info.Digest,
			)
			service.LocalClipboardService.SetInfo(info)
			go func() {
				log.Println("📡 start pushing clipboard to remote...")
				err := service.RemoteClipboardService.PushInfo(rpcClient, info)
				if err != nil {
					log.Printf("❌ failed to push clipboard to remote: %v", err)
				}
				log.Println("✅ successfully pushed clipboard to remote")
			}()
		}
	}
}
