package service

import (
	"hedgex-server/config"
	"hedgex-server/gl"
	"hedgex-server/model"
	"sync"
	"sync/atomic"
	"time"
)

var expUserList map[string]*ExplosiveList //current accounts waiting for be detected to explosive

func init() {
	expUserList = make(map[string]*ExplosiveList)
	for i := range config.Contract {
		expUserList[config.Contract[i]] = NewExplosiveList()
	}
}

//StartExplosiveDetectServer, no blocking function
func StartExplosiveDetectServer() {
	timer := time.NewTicker(config.Explosive.Tick * time.Second)
	for range timer.C {
		for _, contract := range config.Contract {
			//get the current price of contract
			price := atomic.LoadInt64(IndexPrices[contract])
			if price == 0 {
				gl.OutLogger.Warn("Get index price error. %s", contract)
				continue
			}

			//check long position users
			for {
				expUserList[contract].mu.Lock()
				node := expUserList[contract].LHead.Next
				expUserList[contract].mu.Unlock()
				if (node == nil) || (node.ExPrice < price) {
					break
				}
				auth, err := gl.GetAccountAuth()
				if err != nil {
					gl.OutLogger.Error("Get auth error  when explosive user. %v", err)
					break
				}
				if err := gl.Explosive(auth, contract, node.Account); err != nil {
					gl.OutLogger.Error("Explosive error. %s : %s : %v", contract, node.Account, err)
					break
				} else {
					expUserList[contract].Delete(node.Account)
				}
			}

			//check short position users
			for {
				expUserList[contract].mu.Lock()
				node := expUserList[contract].SHead.Next
				expUserList[contract].mu.Unlock()
				if (node == nil) || (node.ExPrice > price) {
					break
				}
				auth, err := gl.GetAccountAuth()
				if err != nil {
					gl.OutLogger.Error("Get auth error  when explosive user. %v", err)
					break
				}
				if err := gl.Explosive(auth, contract, node.Account); err != nil {
					gl.OutLogger.Error("Explosive error. %s : %s : %v", contract, node.Account, err)
					break
				} else {
					expUserList[contract].Delete(node.Account)
				}
			}
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
	LHead *UserNode            // long position user
	SHead *UserNode            // short position user
	Index map[string]*UserNode // the user node's index
	mu    sync.Mutex
}

func NewExplosiveList() *ExplosiveList {
	return &ExplosiveList{
		LHead: &UserNode{},
		SHead: &UserNode{},
		Index: make(map[string]*UserNode),
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
	keepMargin := (u.Lposition*u.Lprice + u.Sposition*u.Sprice) / 30
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
