module github.com/kashalls/waifu/apps/listener

go 1.24.0

require (
	github.com/bwmarrin/discordgo v0.28.1
	github.com/kashalls/waifu/pkg/event v0.0.0-00010101000000-000000000000
	github.com/redis/go-redis/v9 v9.7.0
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	golang.org/x/crypto v0.45.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
)

replace github.com/kashalls/waifu/pkg/event => ../../pkg/event
