package service

import (
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserStore interface {
	Save(user *User) error
	Find(username string) (*User, error)
}

type InMemoryUserStore struct {
	mutex sync.RWMutex
	users map[string]*User
}

func NewInMemoryUserStore() *InMemoryUserStore {
	return &InMemoryUserStore{
		users: make(map[string]*User),
	}
}

func (store *InMemoryUserStore) Save(user *User) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	if store.users[user.Usename] != nil {
		return status.Errorf(codes.AlreadyExists, "the user is already exists")
	}

	store.users[user.Usename] = user.Clone()
	return nil
}

func (store *InMemoryUserStore) Find(username string) (*User, error) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	user := store.users[username]
	if user == nil {
		return nil, nil
	}
	return user, nil

}
