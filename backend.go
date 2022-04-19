package dovecot

type Backend interface {
	Lookup(client Client, path string) (Reply, interface{}, error)
}
