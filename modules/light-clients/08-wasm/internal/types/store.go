package types

import (
	"bytes"
	"errors"
	"io"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"

	"cosmossdk.io/store/cachekv"
	"cosmossdk.io/store/tracekv"
	storetypes "cosmossdk.io/store/types"
)

var (
	_ wasmvmtypes.KVStore = &StoreAdapter{}
	_ storetypes.KVStore  = &MergedClientStore{}

	SubjectPrefix    = []byte("subject/")
	SubstitutePrefix = []byte("substitute/")
)

// MergedClientStore combines two KVStores into one.
//
// Both stores are used for reads, but only the subjectStore is used for writes. For all operations, the key
// is checked to determine which types to use and must be prefixed with either "subject/" or "substitute/" accordingly.
// If the key is not prefixed with either "subject/" or "substitute/", a default action is taken (e.g. no-op for Set/Delete).
type MergedClientStore struct {
	subjectStore    storetypes.KVStore
	substituteStore storetypes.KVStore
}

// NewMergedClientStore retusn a new instance of a MergedClientStore
func NewMergedClientStore(subjectStore, substituteStore storetypes.KVStore) MergedClientStore {
	if subjectStore == nil {
		panic(errors.New("subjectStore must not be nil"))
	}
	if substituteStore == nil {
		panic(errors.New("substituteStore must not be nil"))
	}

	return MergedClientStore{
		subjectStore:    subjectStore,
		substituteStore: substituteStore,
	}
}

// Get implements the storetypes.KVStore interface. It allows reads from both the subjectStore and substituteStore.
//
// Get will return an empty byte slice if the key is not prefixed with either "subject/" or "substitute/".
func (ws MergedClientStore) Get(key []byte) []byte {
	prefix, key := SplitPrefix(key)

	store, found := ws.GetStore(prefix)
	if !found {
		// return a nil byte slice as KVStore.Get() does by default
		return []byte(nil)
	}

	return store.Get(key)
}

// Has implements the storetypes.KVStore interface. It allows reads from both the subjectStore and substituteStore.
//
// Note: contracts do not have access to the Has method, it is only implemented here to satisfy the storetypes.KVStore interface.
func (ws MergedClientStore) Has(key []byte) bool {
	prefix, key := SplitPrefix(key)

	store, found := ws.GetStore(prefix)
	if !found {
		// return false as value when types is not found
		return false
	}

	return store.Has(key)
}

// Set implements the storetypes.KVStore interface. It allows writes solely to the subjectStore.
//
// Set will no-op if the key is not prefixed with "subject/".
func (ws MergedClientStore) Set(key, value []byte) {
	prefix, key := SplitPrefix(key)
	if !bytes.Equal(prefix, SubjectPrefix) {
		return // no-op
	}
	ws.subjectStore.Set(key, value)
}

// Delete implements the storetypes.KVStore interface. It allows deletions solely to the subjectStore.
//
// Delete will no-op if the key is not prefixed with "subject/".
func (ws MergedClientStore) Delete(key []byte) {
	prefix, key := SplitPrefix(key)
	if !bytes.Equal(prefix, SubjectPrefix) {
		return // no-op
	}

	ws.subjectStore.Delete(key)
}

// Iterator implements the storetypes.KVStore interface. It allows iteration over both the subjectStore and substituteStore.
//
// Iterator will return a closed iterator if the start or end keys are not prefixed with either "subject/" or "substitute/".
func (ws MergedClientStore) Iterator(start, end []byte) storetypes.Iterator {
	prefixStart, start := SplitPrefix(start)
	prefixEnd, end := SplitPrefix(end)

	if !bytes.Equal(prefixStart, prefixEnd) {
		return ws.closedIterator()
	}

	store, found := ws.GetStore(prefixStart)
	if !found {
		return ws.closedIterator()
	}

	return store.Iterator(start, end)
}

// ReverseIterator implements the storetypes.KVStore interface. It allows iteration over both the subjectStore and substituteStore.
//
// ReverseIterator will return a closed iterator if the start or end keys are not prefixed with either "subject/" or "substitute/".
func (ws MergedClientStore) ReverseIterator(start, end []byte) storetypes.Iterator {
	prefixStart, start := SplitPrefix(start)
	prefixEnd, end := SplitPrefix(end)

	if !bytes.Equal(prefixStart, prefixEnd) {
		return ws.closedIterator()
	}

	store, found := ws.GetStore(prefixStart)
	if !found {
		return ws.closedIterator()
	}

	return store.ReverseIterator(start, end)
}

// GetStoreType implements the storetypes.KVStore interface, it is implemented solely to satisfy the interface.
func (ws MergedClientStore) GetStoreType() storetypes.StoreType {
	return ws.substituteStore.GetStoreType()
}

// CacheWrap implements the storetypes.KVStore interface, it is implemented solely to satisfy the interface.
func (ws MergedClientStore) CacheWrap() storetypes.CacheWrap {
	return cachekv.NewStore(ws)
}

// CacheWrapWithTrace implements the storetypes.KVStore interface, it is implemented solely to satisfy the interface.
func (ws MergedClientStore) CacheWrapWithTrace(w io.Writer, tc storetypes.TraceContext) storetypes.CacheWrap {
	return cachekv.NewStore(tracekv.NewStore(ws, w, tc))
}

// getStore returns the types to be used for the given key and a boolean flag indicating if that types was found.
// If the key is prefixed with "subject/", the subjectStore is returned. If the key is prefixed with "substitute/",
// the substituteStore is returned.
//
// If the key is not prefixed with either "subject/" or "substitute/", a nil types is returned and the boolean flag is false.
func (ws MergedClientStore) GetStore(prefix []byte) (storetypes.KVStore, bool) {
	if bytes.Equal(prefix, SubjectPrefix) {
		return ws.subjectStore, true
	} else if bytes.Equal(prefix, SubstitutePrefix) {
		return ws.substituteStore, true
	}

	return nil, false
}

// closedIterator returns an iterator that is always closed, used when Iterator() or ReverseIterator() is called
// with an invalid prefix or start/end key.
func (ws MergedClientStore) closedIterator() storetypes.Iterator {
	// Create a dummy iterator that is always closed right away.
	it := ws.subjectStore.Iterator([]byte{0}, []byte{1})
	it.Close()

	return it
}

// SplitPrefix splits the key into the prefix and the key itself, if the key is prefixed with either "subject/" or "substitute/".
// If the key is not prefixed with either "subject/" or "substitute/", the prefix is nil.
func SplitPrefix(key []byte) ([]byte, []byte) {
	if bytes.HasPrefix(key, SubjectPrefix) {
		return SubjectPrefix, bytes.TrimPrefix(key, SubjectPrefix)
	} else if bytes.HasPrefix(key, SubstitutePrefix) {
		return SubstitutePrefix, bytes.TrimPrefix(key, SubstitutePrefix)
	}

	return nil, key
}

// StoreAdapter bridges the SDK types implementation to wasmvm one. It implements the wasmvmtypes.KVStore interface.
type StoreAdapter struct {
	parent storetypes.KVStore
}

// NewStoreAdapter constructor
func NewStoreAdapter(s storetypes.KVStore) *StoreAdapter {
	if s == nil {
		panic(errors.New("types must not be nil"))
	}
	return &StoreAdapter{parent: s}
}

// Get implements the wasmvmtypes.KVStore interface.
func (s StoreAdapter) Get(key []byte) []byte {
	return s.parent.Get(key)
}

// Set implements the wasmvmtypes.KVStore interface.
func (s StoreAdapter) Set(key, value []byte) {
	s.parent.Set(key, value)
}

// Delete implements the wasmvmtypes.KVStore interface.
func (s StoreAdapter) Delete(key []byte) {
	s.parent.Delete(key)
}

// Iterator implements the wasmvmtypes.KVStore interface.
func (s StoreAdapter) Iterator(start, end []byte) wasmvmtypes.Iterator {
	return s.parent.Iterator(start, end)
}

// ReverseIterator implements the wasmvmtypes.KVStore interface.
func (s StoreAdapter) ReverseIterator(start, end []byte) wasmvmtypes.Iterator {
	return s.parent.ReverseIterator(start, end)
}
