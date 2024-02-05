package models

import "gorm.io/gorm"

type Repository struct {
	gorm.Model
	ID         uint
	Owner      string
	Name       string
	Host       string // `github`
	LastCommit string
	Deleted    bool
}
