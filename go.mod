module github.com/dextryz/ixian

go 1.21.0

require (
	github.com/dextryz/nostr v0.2.0
	github.com/gomarkdown/markdown v0.0.0-20230922112808-5421fefb8386
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.5.0
	github.com/mattn/go-sqlite3 v1.14.18
)

require (
	github.com/btcsuite/btcd/btcec/v2 v2.3.2 // indirect
	github.com/btcsuite/btcd/btcutil v1.1.3 // indirect
	github.com/btcsuite/btcd/chaincfg/chainhash v1.0.1 // indirect
	github.com/decred/dcrd/crypto/blake256 v1.0.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
)

replace github.com/dextryz/nostr => ../nostr
