module github.com/kashalls/waifu/apps/ingestor

go 1.24

require (
	github.com/kashalls/waifu/pkg/event v0.0.0-00010101000000-000000000000
	github.com/redis/go-redis/v9 v9.7.3
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
)

replace github.com/kashalls/waifu/pkg/event => ../../pkg/event
