package dovecot

type Backend interface {
	Lookup(client Client, dict string, search string) (string, error)
}
