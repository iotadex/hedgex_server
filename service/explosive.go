package service

import (
	"hedgex-server/config"
	"hedgex-server/gl"
	"hedgex-server/model"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

var expUserList map[string]*ExplosiveList //current accounts waiting for be detected to explosive

func init() {
	expUserList = make(map[string]*ExplosiveList)
	for i := range config.Contract {
		scale := gl.KeepMarginScale[config.Contract[i]]
		expUserList[config.Contract[i]] = NewExplosiveList(scale)
	}
}

//StartExplosiveDetectServer, no blocking function
func StartExplosiveDetect() {
	for _, contract := range config.Contract {
		go runExplosiveDetect(contract, expUserList[contract])
	}
}

type ExpTx struct {
	tx      *types.Transaction
	account string
}

func runExplosiveDetect(conAdd string, el *ExplosiveList) {
	txes := make([]ExpTx, 0)
	timer := time.NewTicker(config.Explosive.Tick * time.Second)
	for range timer.C {
		pendingTx := make([]ExpTx, 0)
		// check txes from chain
		for i := range txes {
			if receipt, err := gl.GetTransaction(txes[i].tx.Hash()); err != nil {
				gl.OutLogger.Info("Get tx(%s) receipt error. %v", txes[i].tx.Hash().Hex(), err)
				pendingTx = append(pendingTx, txes[i])
			} else {
				gl.OutLogger.Info("Explosive accept. %d : %s", receipt.Status, receipt.TxHash.Hex())
			}
		}
		txes = pendingTx
		if len(txes) > 0 {
			continue
		}

		//get the current price of contract
		price, err := gl.GetIndexPrice(conAdd)
		if err != nil {
			gl.OutLogger.Error("Get price from contract error. %s : %v", conAdd, err)
			continue
		}

		// get the accounts which need to be explosived
		expAccounts := make([]string, 0)
		el.mu.Lock()
		node := el.LHead.Next
		for node != nil {
			if node.ExPrice < price {
				break
			}
			expAccounts = append(expAccounts, node.Account)
			node = node.Next
		}
		node = el.SHead.Next
		for node != nil {
			if node.ExPrice > price {
				break
			}
			expAccounts = append(expAccounts, node.Account)
			node = node.Next
		}
		el.mu.Unlock()

		for _, account := range expAccounts {
			err = gl.GetGasPriceAndNonce(config.Explosive.GasPriceMin, gl.ExplosiveAuth, gl.ExplosivePA)
			if err != nil {
				gl.OutLogger.Error("Get auth error when explosive user. %v", err)
				continue
			}
			tx, err := gl.Explosive(gl.ExplosiveAuth, conAdd, account)
			if err != nil {
				gl.OutLogger.Error("Explosive error. %s : %s : %v", conAdd, account, err)
				continue
			}
			txes = append(txes, ExpTx{tx: tx, account: account})
			time.Sleep(time.Millisecond * 200)
		}
	}
}

type UserNode struct {
	model.User
	ExPrice int64 // user's explosive price
	Pre     *UserNode
	Next    *UserNode
}

type ExplosiveList struct {
	LHead           *UserNode            // long position user
	SHead           *UserNode            // short position user
	Index           map[string]*UserNode // the user node's index
	KeepMarginScale uint64
	mu              sync.Mutex
}

func NewExplosiveList(scale uint64) *ExplosiveList {
	return &ExplosiveList{
		LHead:           &UserNode{},
		SHead:           &UserNode{},
		Index:           make(map[string]*UserNode),
		KeepMarginScale: scale,
	}
}

func (el *ExplosiveList) Insert(u *model.User) {
	if (u == nil) || (u.Lposition == u.Sposition) {
		return
	}
	el.mu.Lock()
	defer el.mu.Unlock()
	if _, exist := el.Index[u.Account]; exist {
		return
	}
	keepMargin := (u.Lposition*u.Lprice + u.Sposition*u.Sprice) / el.KeepMarginScale
	ePrice := (int64(keepMargin) - u.Margin + int64(u.Lposition*u.Lprice) - int64(u.Sposition*u.Sprice)) / (int64(u.Lposition) - int64(u.Sposition))
	var currNode *UserNode
	node := &UserNode{
		User:    *u,
		ExPrice: ePrice,
	}
	if u.Lposition > u.Sposition {
		// find the first node that ExPrice < ePrice
		currNode = el.LHead
		for {
			if (currNode.Next == nil) || (currNode.Next.ExPrice < ePrice) {
				break
			}
			currNode = currNode.Next
		}
	} else {
		currNode = el.SHead
		for {
			if (currNode.Next == nil) || (currNode.Next.ExPrice > ePrice) {
				break
			}
			currNode = currNode.Next
		}
	}
	node.Pre = currNode
	if currNode.Next != nil {
		node.Next = currNode.Next
		currNode.Next.Pre = node
	}
	currNode.Next = node
	el.Index[u.Account] = node
}

func (el *ExplosiveList) Delete(account string) {
	el.mu.Lock()
	defer el.mu.Unlock()
	node, exist := el.Index[account]
	if !exist {
		return
	}
	if node.Next != nil {
		node.Next.Pre = node.Pre
	}
	if node.Pre != nil {
		node.Pre.Next = node.Next
	}
	if node.Next == nil && node.Pre == nil {
		el.LHead = nil
		el.SHead = nil
	}
	delete(el.Index, account)
}

func (el *ExplosiveList) Update(u *model.User) {
	if u == nil {
		return
	}
	el.mu.Lock()
	if user, exist := el.Index[u.Account]; exist {
		if user.Block > u.Block {
			el.mu.Unlock()
			return
		}
	}
	el.mu.Unlock()
	el.Delete(u.Account)
	el.Insert(u)
}
