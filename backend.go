package dovecot

type Backend interface {
	Lookup(client Client, keyType string, key string, namespace string) (Reply, string, error)
}
