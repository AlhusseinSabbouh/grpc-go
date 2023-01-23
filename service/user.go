package service

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Usename        string
	HashedPassword string
	Role           string
}

func NewUser(username, password, role string) (*User, error) {
	HashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("cannot hash the password, %w", err)
	}
	user := &User{
		Usename:        username,
		HashedPassword: string(HashedPassword),
		Role:           role,
	}
	return user, nil
}

func (user *User) IsCorrectPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(password))
	return err == nil
}

func (user *User) Clone() *User {
	return &User{
		Usename:        user.Usename,
		HashedPassword: user.HashedPassword,
		Role:           user.Role,
	}
}
