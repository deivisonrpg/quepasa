package models

type QpDataUserContextsInterface interface {
	Find(username string, contextID string) (*QpUserContextAccess, error)
	ListAll() ([]*QpUserContextAccess, error)
	ListForUser(username string, enabledOnly bool) ([]*QpUserContextAccess, error)
	Upsert(access *QpUserContextAccess) error
	Delete(username string, contextID string) (bool, error)
}
