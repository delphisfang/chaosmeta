/*
 * Copyright 2022-2023 Chaos Meta Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package user

import (
	namespace2 "chaosmeta-platform/pkg/models/namespace"
	"chaosmeta-platform/pkg/models/user"
	"chaosmeta-platform/pkg/service/namespace"
	"chaosmeta-platform/util/errors"
	"chaosmeta-platform/util/log"
	"context"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"time"
)

type GrantType string

var (
	GrantTypeAccess  GrantType = "access"
	GrantTypeRefresh GrantType = "refresh"
	Admin                      = "admin"
)

type UserRole string

const (
	NormalRole = UserRole("normal")
	AdminRole  = UserRole("admin")
)

func Init() {
	us := UserService{}
	ctx := context.Background()

	_, err := us.Get(ctx, "admin")
	if err == nil {
		log.Error(err)
		return
	}
	_, err = us.Create(ctx, "admin", "admin", string(AdminRole))
	if err != nil {
		log.Error(err)
	}
}

type UserService struct{}

func (a *UserService) InitAdmin(ctx context.Context, name, password string) error {
	user, err := a.Get(ctx, Admin)
	if err == nil && user != nil {
		return nil
	}
	_, err = a.Create(ctx, Admin, Admin, string(AdminRole))
	return err
}

func (a *UserService) IsAdmin(ctx context.Context, name string) bool {
	userGet := user.User{Email: name}
	if err := user.GetUser(ctx, &userGet); err != nil {
		return false
	}
	if userGet.Disabled {
		return false
	}
	return userGet.Role == user.AdminRole
}

func (a *UserService) Login(ctx context.Context, name, password string) (string, string, error) {
	userGet := user.User{Email: name}
	if err := user.GetUser(ctx, &userGet); err != nil {
		return "", "", err
	}
	if userGet.Disabled || userGet.IsDeleted {
		return "", "", errors.ErrUnauthorized()
	}
	if !VerifyPassword(password, userGet.Password) {
		return "", "", errors.ErrUnauthorized()
	}

	userGet.LastLoginTime = time.Now()
	if err := user.UpdateUser(ctx, &userGet); err != nil {
		return "", "", err
	}

	authentication := Authentication{}
	tocken, err := authentication.GenerateToken(name, string(GrantTypeAccess), 5*time.Minute)
	if err != nil {
		return "", "", err
	}

	refreshToken, err := authentication.GenerateToken(name, string(GrantTypeRefresh), time.Hour*24)
	if err != nil {
		return "", "", err
	}
	return tocken, refreshToken, nil
}

func (a *UserService) Create(ctx context.Context, name, password, role string) (int, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return 0, err
	}

	userGet, err := a.Get(ctx, name)
	if err == nil && userGet.IsDeleted {
		userGet.IsDeleted = false
		userGet.Password = hash
		return userGet.ID, user.UpdateUser(ctx, userGet)
	}

	userCreate := user.User{
		Email:     name,
		Password:  hash,
		Role:      role,
		Disabled:  false,
		IsDeleted: false,
	}

	_, err = user.InsertUser(ctx, &userCreate)
	if err != nil {
		return 0, err
	}

	us := namespace.NamespaceService{}
	return userCreate.ID, us.DefaultAddUsers(ctx, namespace2.AddUsersParam{
		Users: []namespace2.UserData{{
			Id:         userCreate.ID,
			Permission: int(namespace2.AdminPermission),
		}},
	})
}

func (a *UserService) Get(ctx context.Context, name string) (*user.User, error) {
	userGet := user.User{Email: name}
	if err := user.GetUser(ctx, &userGet); err != nil {
		return nil, err
	}
	return &userGet, nil
}

func (a *UserService) GetList(ctx context.Context, name, role, orderBy string, page, pageSize int) (int64, []user.User, error) {
	return user.QueryUser(ctx, name, role, orderBy, page, pageSize)
}

func (a *UserService) GetListWithNamespaceInfo(ctx context.Context, namespaceId int, name, role, orderBy string, page, pageSize int) (int64, []namespace.UserInfoInNamespace, error) {
	total, userList, err := user.QueryUser(ctx, name, role, orderBy, page, pageSize)
	if err != nil {
		return 0, nil, err
	}
	ns := namespace.NamespaceService{}
	userInfoList, err := ns.GetUsersOfNamespacePermissions(ctx, userList, namespaceId)
	return total, userInfoList, err
}

func (a *UserService) GetNamespaceList(ctx context.Context, name string, permission int, orderBy string, page, pageSize int) (int64, []namespace2.UserNamespaceData, error) {
	userGet, err := a.Get(ctx, name)
	if err != nil {
		return 0, nil, err
	}
	return namespace2.GetNamespacesFromUser(ctx, userGet.ID, permission, orderBy, page, pageSize)
}

func (a *UserService) DeleteList(ctx context.Context, name string, deleteIds []int) error {
	if !a.IsAdmin(ctx, name) {
		return fmt.Errorf("not admin")
	}

	if err := user.DeleteUsersByIdList(ctx, deleteIds); err != nil {
		return err
	}
	return namespace2.UsersOrNamespacesDelete(deleteIds, nil)
}

func (a *UserService) UpdatePassword(ctx context.Context, name, newPassword string) error {
	userGet, err := a.Get(ctx, name)
	if err != nil {
		return err
	}
	if userGet.Disabled {
		return errors.ErrUnauthorized()
	}
	if !a.IsAdmin(ctx, name) && name != userGet.Email {
		return errors.ErrUnauthorized()
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	userGet.Password = hash
	return user.UpdateUser(ctx, userGet)
}

func (a *UserService) UpdateListRole(ctx context.Context, name string, ids []int, role string) error {
	if !a.IsAdmin(ctx, name) {
		return fmt.Errorf("not admin")
	}

	return user.UpdateUsersRole(ctx, ids, role)
}

func (a *UserService) UpdateListNamespace(ctx context.Context, name string, ids []int, role string) error {
	if !a.IsAdmin(ctx, name) {
		return fmt.Errorf("not admin")
	}

	return user.UpdateUsersRole(ctx, ids, role)
}

func (a *UserService) CheckToken(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", errors.ErrUnauthorized()
	}
	authentication := Authentication{}
	tokenClaims, err := authentication.VerifyToken(token)
	if err != nil {
		return "", errors.ErrUnauthorized()
	}
	if tokenClaims.GrantType != string(GrantTypeAccess) {
		return "", errors.ErrUnauthorized()
	}
	return tokenClaims.Username, nil
}

func (a *UserService) RefreshToken(ctx context.Context, token string) (string, error) {
	authentication := Authentication{}
	return authentication.RefreshToken(token, string(GrantTypeAccess))
}

// Generate a user's hashed password
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Verify that the user's password is correct
func VerifyPassword(password string, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	fmt.Println(err)
	return err == nil
}
