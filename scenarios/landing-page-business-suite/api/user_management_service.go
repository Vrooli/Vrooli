package main

import "landing-page-business-suite-api/internal/account"

type UserManagementStore = account.UserManagementStore
type UserManagementService = account.UserManagementService

func NewUserManagementService(db UserManagementStore) *UserManagementService {
	return account.NewUserManagementService(db)
}
