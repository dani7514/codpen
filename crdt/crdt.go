package crdt

type CRDT interface {
	Insert(position int, value string) (string, error)
	Delete(position int) string
}
