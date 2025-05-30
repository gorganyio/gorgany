package core

type IDomain[T any] interface {
	Query() IOrm[T]
	Clone() *T
	GetDomainMeta() IDomainMeta
}

type IDomainMeta interface {
	SetLoaded(loaded bool)
	SetTable(table string)
	SetDriver(driver DbType)
	SetOriginal(original any)
	SetDomain(domain any)
}

type IFile interface {
	FullPath() string
	PublicPath() string
	PathInPublic() string
	ReadContent() (string, error)
	Save(p string) error
	IsExists() bool
	IsNil() bool
	Delete() error
}

type ILocalizedString interface {
	Text(lang string) string
	Map() map[string]string
}

type ProtectedFields interface {
	GetProtectedFields() []string
}
