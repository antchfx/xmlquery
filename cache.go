package xmlquery

import (
	"fmt"
	"sync"

	"github.com/antchfx/xpath"
	"github.com/golang/groupcache/lru"
)

// DisableSelectorCache will disable caching for the query selector if value is true.
var DisableSelectorCache = false

// SelectorCacheMaxEntries allows how many selector object can be caching. Default is 50.
// Will disable caching if SelectorCacheMaxEntries <= 0.
var SelectorCacheMaxEntries = 50

var (
	cacheOnce  sync.Once
	cache      *lru.Cache
	cacheMutex sync.Mutex
)

func getQuery(expr string, opts xpath.CompileOptions) (*xpath.Expr, error) {
	key := expr + fmt.Sprintf("%#v", opts)
	if DisableSelectorCache || SelectorCacheMaxEntries <= 0 {
		return xpath.CompileWithOptions(expr, opts)
	}
	cacheOnce.Do(func() {
		cache = lru.New(SelectorCacheMaxEntries)
	})
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	if v, ok := cache.Get(key); ok {
		return v.(*xpath.Expr), nil
	}
	v, err := xpath.CompileWithOptions(expr, opts)
	if err != nil {
		return nil, err
	}
	cache.Add(key, v)
	return v, nil
}
