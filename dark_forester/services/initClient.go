package services

import (
	"fmt"
	"log"
	"reflect"
	"unsafe"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	// [kyzooghost] Ahh interesting, can't stream pending tx with public endpoint
	// public bsc endpoint. You can't stream pending tx with those.
	bsc_testnet = "https://data-seed-prebsc-2-s1.binance.org:8545/"
	bsc         = "https://bsc-dataseed.binance.org/"
	// geth AWS server
	geth_http = "http://x.xxx.xxx.xxx:8545"
	geth_ipc  = "/home/ubuntu/bsc/node/geth.ipc"
)

var ClientEntered *string

func GetCurrentClient() *ethclient.Client {

	var clientType string

	switch *ClientEntered {
	case "bsc_testnet":
		clientType = bsc_testnet
	case "bsc":
		clientType = bsc
	case "geth_http":
		clientType = geth_http
	default:
		clientType = geth_ipc
	}

	// [kyzooghost] Create new client connection to an Ethereum node
	// [kyzooghost] Can take HTTP or IPC endpoint
	// [kyzooghost] Interprocess communication (IPC) allow processes on same machine to communicate. Connect GETH node to Unix domain socket or named pipe. Typically faster and more efficient than HTTP or WebSocket connections.
	client, err := ethclient.Dial(clientType)

	if err != nil {
		fmt.Println("Error connecting to client", clientType)
		log.Fatalln(err)
	} else {
		fmt.Println("Successffully connected to ", clientType)
	}

	return client
}

// [kyzooghost] Use reflection to access private field at runtime?
func InitRPCClient(_ClientEntered *string) *rpc.Client {

	ClientEntered = _ClientEntered
	var clientValue reflect.Value
	// [kyzooghost] Get value (?actual ethclient.Client object)
	clientValue = reflect.ValueOf(GetCurrentClient()).Elem()
	// [kyzooghost] Can't we just access obj.c directly via dot notation?
	// [kyzooghost] *ethclient.Client is a wrapper for *rpc.Client, with higher-level blockchain-specific methods
	// [kyzooghost] *rpc.Client is generic client for making RPC calls
	fieldStruct := clientValue.FieldByName("c")
	// [kyzooghost] Create new value of type 'fieldStruct' or *rpc.Client
	// [kyzooghost] Pointer at fieldStruct address
	// [kyzooghost] ?Reinitialize fieldStruct object?
	clientPointer := reflect.NewAt(fieldStruct.Type(), unsafe.Pointer(fieldStruct.UnsafeAddr())).Elem()
	// [kyzooghost] .Interface() -> Convert reflection value to interface
	// [kyzooghost] .(*rpc.Client) -> Assert interface type
	finalClient, _ := clientPointer.Interface().(*rpc.Client)
	return finalClient
}
