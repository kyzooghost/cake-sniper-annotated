package main

import (
	"context"
	"dark_forester/contracts/erc20"
	"dark_forester/global"
	"dark_forester/services"
	"flag"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// Entry point of dark_forester.
// Before Anything, check /global/config to correctly parametrize the bot

// [kyzooghost] Empty collection of goroutines - collection of concurrent tasks
var wg = sync.WaitGroup{}

// [kyzooghost] Channels used to communicate between goroutines
var TopSnipe = make(chan *big.Int)

// convert WEI to ETH
func formatEthWeiToEther(etherAmount *big.Int) float64 {
	var base, exponent = big.NewInt(10), big.NewInt(18)
	denominator := base.Exp(base, exponent, nil)
	tokensSentFloat := new(big.Float).SetInt(etherAmount)
	denominatorFloat := new(big.Float).SetInt(denominator)
	final, _ := new(big.Float).Quo(tokensSentFloat, denominatorFloat).Float64()
	return final
}

// fetch ERC20 token symbol
func getTokenSymbol(tokenAddress common.Address, client *ethclient.Client) string {
	tokenIntance, _ := erc20.NewErc20(tokenAddress, client)
	sym, _ := tokenIntance.Symbol(nil)
	return sym
}

// main loop of the bot
func StreamNewTxs(client *ethclient.Client, rpcClient *rpc.Client) {

	// Go channel to pipe data from client subscription
	newTxsChannel := make(chan common.Hash)

	// Subscribe to receive one time events for new txs
	// [kyzooghost] Create subscription, get subscribed 'newPendingTransactions' events from channel
	_, err := rpcClient.EthSubscribe(
		context.Background(), newTxsChannel, "newPendingTransactions", // no additional args
	)

	/**
	 * [kyzooghost] context.Context to manage long-running or async tasks
	 * - Propagate cancellation request across goroutines
	 * - Specify deadline or timeout
	 * - Pass request-scoped values
	 */

	if err != nil {
		fmt.Println("error while subscribing: ", err)
	}
	fmt.Println("\nSubscribed to mempool txs!\n")

	fmt.Println("\n////////////// BIG TRANSFERS //////////////////\n")
	if global.BIG_BNB_TRANSFER == true {
		fmt.Println("activated\nthreshold of interest : transfers >", global.BNB[:2], " BNB")
	} else {
		fmt.Println("not activated")
	}

	fmt.Println("\n////////////// ADDRESS MONITORING //////////////////\n")
	if global.ADDRESS_MONITOR == true {
		fmt.Println("activated\nthe following addresses are monitored : \n")
		for addy, addressData := range global.AddressesWatched {
			fmt.Println("address : ", addy, "name: ", addressData.Name)
		}
	} else {
		fmt.Println("not activated")
	}

	fmt.Println("\n////////////// SANDWICHER //////////////////\n")
	if global.Sandwicher == true {
		fmt.Println("activated\n\nmax BNB amount authorised for one sandwich : ", global.Sandwicher_maxbound, "WBNB")
		fmt.Println("minimum profit expected : ", global.Sandwicher_minprofit, "WBNB")
		fmt.Println("current WBNB balance inside TRIGGER : ", formatEthWeiToEther(global.GetTriggerWBNBBalance()), "WBNB")
		fmt.Println("TRIGGER balance at which we stop execution : ", formatEthWeiToEther(global.STOPLOSSBALANCE), "WBNB")
		fmt.Println("WARNING: be sure TRIGGER WBNB balance is > SANDWICHER MAXBOUND !!")

		activeMarkets := 0
		for _, specs := range global.SANDWICH_BOOK {
			if specs.Whitelisted == true && specs.ManuallyDisabled == false {
				// fmt.Println(specs.Name, market, specs.Liquidity)
				activeMarkets += 1
			}
		}
		fmt.Println("\nNumber of active Markets: ", activeMarkets, "\n")

		fmt.Println("\nManually disabled Markets: \n")
		for market, specs := range global.SANDWICH_BOOK {
			if specs.ManuallyDisabled == true {
				fmt.Println(specs.Name, market, specs.Liquidity)
			}
		}
		fmt.Println("\nEnnemies: \n")
		for ennemy, _ := range global.ENNEMIES {
			fmt.Println(ennemy)
		}

	} else {
		fmt.Println("not activated")
	}

	fmt.Println("\n////////////// LIQUIDITY SNIPING //////////////////\n")
	if global.Sniping == true {
		fmt.Println("activated")
		name, _ := global.Snipe.Tkn.Name(&bind.CallOpts{})
		fmt.Println("token targetted: ", global.Snipe.TokenAddress, "(", name, ")")
		fmt.Println("minimum liquidity expected : ", formatEthWeiToEther(global.Snipe.MinLiq), getTokenSymbol(global.Snipe.TokenPaired, client))
		fmt.Println("current WBNB balance inside TRIGGER : ", formatEthWeiToEther(global.GetTriggerWBNBBalance()), "WBNB")

	} else {
		fmt.Println("not activated")
	}
	chainID, _ := client.NetworkID(context.Background())
	// [kyzooghost] EIP155 add chain-specific ID to prevent replay attack across chains
	signer := types.NewEIP155Signer(chainID)

	for {
		select {
		// Code block is executed when a new tx hash is piped to the channel
		case transactionHash := <-newTxsChannel:
			// Get transaction object from hash by querying the client
			tx, is_pending, _ := client.TransactionByHash(context.Background(), transactionHash)
			// If tx is valid and still unconfirmed
			if is_pending {
				// [kyzooghost] If we ditch the return values, what is the point of this call?
				_, _ = signer.Sender(tx)
				handleTransaction(tx, client)
			}
		}
	}
}

func handleTransaction(tx *types.Transaction, client *ethclient.Client) {
	services.TxClassifier(tx, client, TopSnipe)
}

func main() {
	// [kyzooghost] Parse CLI argument
	// we say <place_holder> for the defval as it is anyway filtered to geth_ipc in the switch
	ClientEntered := flag.String("client", "xxx", "Gateway to the bsc protocol. Available options:\n\t-bsc_testnet\n\t-bsc\n\t-geth_http\n\t-geth_ipc")
	flag.Parse()

	// [kyzooghost] Why don't we just create the *eth.Client object, and access *rpc.Client property on it?
	// [kyzooghost] I don't understand this reflection use, and we are also calling GetCurrentClient() twice?
	rpcClient := services.InitRPCClient(ClientEntered)
	client := services.GetCurrentClient()

	// [kyzooghost] Provide initialized environment variables in 'global' object
	global.InitDF(client)

	// init goroutine Clogg if global.Sniping == true
	if global.Sniping == true {
		// [kyzooghost] Call before starting a goroutine to indicate you are waiting for it to complete
		wg.Add(1)
		// [kyzooghost] Define new goroutine with anonymous function
		go func() {
			services.Clogg(client, TopSnipe)
			// [kyzooghost] call `defer wg.Done()` within goroutine when completed
			wg.Done()
			// [kyzooghost] Syntax for immediately invoked
		}()
	}

	// Launch txpool streamer
	StreamNewTxs(client, rpcClient)

}
