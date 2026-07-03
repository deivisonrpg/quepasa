package models

type QpDataSpamSectionsInterface interface {
	Find(token string) (*QpSpamSection, error)
	ListAll() ([]*QpSpamSection, error)
	Upsert(section *QpSpamSection) error
	UpdatePriority(token string, priority int) error
	Delete(token string) (bool, error)
	NextPriority() (int, error)
}
