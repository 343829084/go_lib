package pool

import (
	"go_lib"
	"runtime"
	"time"
)

type InitFunc func() (interface{}, error)

type Pool struct {
	Id        string
	Size      int
	container chan interface{}
	rwSign    go_lib.RWSign
}

func (self *Pool) Init(initFunc InitFunc) error {
	if cap(self.container) != self.Size {
		go_lib.LogInfof("Initializing pool (Id=%v, Size=%v)...\n", self.Id, self.Size)
		self.container = make(chan interface{}, self.Size)
	}
	for i := 0; i < self.Size; i++ {
		element, err := initFunc()
		if err != nil {
			return err
		}
		if element == nil {
			go_lib.LogWarnf("The initialized element is NIL! (poolId=%s)", self.Id)
		}
		self.container <- element
	}
	go_lib.LogInfof("The pool (Id=%v, Size=%v) has been initialized.\n", self.Id, self.Size)
	return nil
}

func (self *Pool) Get(timeoutMs time.Duration) (element interface{}, ok bool) {
	// LogInfof("Getting! (Size: %v, Cap: %v)", len(self.container), cap(self.container))
	if self.Closed() {
		return nil, false
	}
	if timeoutMs > 0 {
		select {
		case element, ok = <-self.container:
			return
		case <-time.After(5 * time.Millisecond):
			go_lib.LogInfof("Getting Timeout! (Size: %v, Cap: %v)", len(self.container), cap(self.container))
			element, ok = nil, false
			return
		}
	} else {
		if len(self.container) == 0 {
			element, ok = nil, false
		} else {
			element, ok = <-self.container
		}
	}
	return
}

func (self *Pool) Put(element interface{}, timeoutMs time.Duration) bool {
	// LogInfof("Putting! (Size: %v, Cap: %v)", len(self.container), cap(self.container))
	if self.Closed() {
		return false
	}
	result := false
	if timeoutMs > 0 {
		sign := make(chan bool)
		go func() {
			time.AfterFunc(5*time.Millisecond, func() {
				if !result {
					go_lib.LogInfof("Putting Timeout! (Size: %v, Cap: %v, Element: %v)", len(self.container), cap(self.container), element)
					sign <- result
				}
				runtime.Goexit()
			})
			self.container <- element
			result = true
			sign <- result
		}()
		return <-sign
	} else {
		if len(self.container) >= self.Size {
			result = false
		} else {
			self.container <- element
			result = true
		}
	}
	return result
}

func (self *Pool) Close() {
	close(self.container)
	self.container = nil
	return
}

func (self *Pool) Closed() bool {
	if self == nil || self.container == nil {
		return true
	}
	return false
}