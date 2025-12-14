package blockchain

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Client struct {
	ethClient    *ethclient.Client
	privateKey   *ecdsa.PrivateKey
	contractAddr common.Address
	chainID      *big.Int
	contractABI  abi.ABI
}

func NewClient(rpcUrl, privKeyHex, contractAddrHex string) (*Client, error) {
	client, err := ethclient.Dial(rpcUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to eth client: %v", err)
	}

	// Remove 0x prefix if present
	privKeyHex = strings.TrimPrefix(privKeyHex, "0x")
	privateKey, err := crypto.HexToECDSA(privKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain id: %v", err)
	}

	contractAddress := common.HexToAddress(contractAddrHex)

	return &Client{
		ethClient:    client,
		privateKey:   privateKey,
		contractAddr: contractAddress,
		chainID:      chainID,
	}, nil
}

func (c *Client) MintToken(toAddrHex string, amount int64) (string, error) {
	// 1. Read ABI
	abiFile, err := os.ReadFile("./blockchain/artifacts/contracts/GridToken.sol/GridToken.json")
	if err != nil {
		return "", fmt.Errorf("failed to read abi file: %v", err)
	}

	var artifact struct {
		Abi interface{} `json:"abi"`
	}
	if err := json.Unmarshal(abiFile, &artifact); err != nil {
		return "", fmt.Errorf("failed to parse artifact json: %v", err)
	}

	abiJSON, err := json.Marshal(artifact.Abi)
	if err != nil {
		return "", fmt.Errorf("failed to marshal abi: %v", err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(string(abiJSON)))
	if err != nil {
		return "", fmt.Errorf("failed to parse abi: %v", err)
	}

	// 2. Bind Contract
	contract := bind.NewBoundContract(c.contractAddr, parsedABI, c.ethClient, c.ethClient, c.ethClient)

	// 3. Prepare Auth
	auth, err := bind.NewKeyedTransactorWithChainID(c.privateKey, c.chainID)
	if err != nil {
		return "", fmt.Errorf("failed to create transactor: %v", err)
	}

	// 4. Transact
	toAddress := common.HexToAddress(toAddrHex)
	
	// Amount * 10^18
	amountBig := big.NewInt(amount)
	exp := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	amountBig.Mul(amountBig, exp)

	tx, err := contract.Transact(auth, "mintReward", toAddress, amountBig)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %v", err)
	}

	return tx.Hash().Hex(), nil
}
