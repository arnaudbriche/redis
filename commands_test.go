package redis_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"gopkg.in/redis.v3"
)

var _ = Describe("Commands", func() {
	var client *redis.Client

	BeforeEach(func() {
		client = redis.NewClient(&redis.Options{
			Addr:         redisAddr,
			ReadTimeout:  500 * time.Millisecond,
			WriteTimeout: 500 * time.Millisecond,
			PoolTimeout:  30 * time.Second,
		})
	})

	AfterEach(func() {
		Expect(client.FlushDb().Err()).NotTo(HaveOccurred())
		Expect(client.Close()).NotTo(HaveOccurred())
	})

	//------------------------------------------------------------------------------

	Describe("server", func() {

		It("should Auth", func() {
			auth := client.Auth("password")
			Expect(auth.Err()).To(MatchError("ERR Client sent AUTH, but no password is set"))
			Expect(auth.Val()).To(Equal(""))
		})

		It("should Echo", func() {
			echo := client.Echo("hello")
			Expect(echo.Err()).NotTo(HaveOccurred())
			Expect(echo.Val()).To(Equal("hello"))
		})

		It("should Ping", func() {
			ping := client.Ping()
			Expect(ping.Err()).NotTo(HaveOccurred())
			Expect(ping.Val()).To(Equal("PONG"))
		})

		It("should Select", func() {
			sel := client.Select(1)
			Expect(sel.Err()).NotTo(HaveOccurred())
			Expect(sel.Val()).To(Equal("OK"))
		})

		It("should BgRewriteAOF", func() {
			r := client.BgRewriteAOF()
			Expect(r.Err()).NotTo(HaveOccurred())
			Expect(r.Val()).To(ContainSubstring("Background append only file rewriting"))
		})

		It("should BgSave", func() {
			// workaround for "ERR Can't BGSAVE while AOF log rewriting is in progress"
			Eventually(func() string {
				return client.BgSave().Val()
			}, "10s").Should(Equal("Background saving started"))
		})

		It("should ClientKill", func() {
			r := client.ClientKill("1.1.1.1:1111")
			Expect(r.Err()).To(MatchError("ERR No such client"))
			Expect(r.Val()).To(Equal(""))
		})

		It("should ClientPause", func() {
			err := client.ClientPause(time.Second).Err()
			Expect(err).NotTo(HaveOccurred())

			Consistently(func() error {
				return client.Ping().Err()
			}, "400ms").Should(HaveOccurred()) // pause time - read timeout

			Eventually(func() error {
				return client.Ping().Err()
			}, "1s").ShouldNot(HaveOccurred())
		})

		It("should ConfigGet", func() {
			r := client.ConfigGet("*")
			Expect(r.Err()).NotTo(HaveOccurred())
			Expect(r.Val()).NotTo(BeEmpty())
		})

		It("should ConfigResetStat", func() {
			r := client.ConfigResetStat()
			Expect(r.Err()).NotTo(HaveOccurred())
			Expect(r.Val()).To(Equal("OK"))
		})

		It("should ConfigSet", func() {
			configGet := client.ConfigGet("maxmemory")
			Expect(configGet.Err()).NotTo(HaveOccurred())
			Expect(configGet.Val()).To(HaveLen(2))
			Expect(configGet.Val()[0]).To(Equal("maxmemory"))

			configSet := client.ConfigSet("maxmemory", configGet.Val()[1].(string))
			Expect(configSet.Err()).NotTo(HaveOccurred())
			Expect(configSet.Val()).To(Equal("OK"))
		})

		It("should DbSize", func() {
			dbSize := client.DbSize()
			Expect(dbSize.Err()).NotTo(HaveOccurred())
			Expect(dbSize.Val()).To(Equal(int64(0)))
		})

		It("should Info", func() {
			info := client.Info()
			Expect(info.Err()).NotTo(HaveOccurred())
			Expect(info.Val()).NotTo(Equal(""))
		})

		It("should LastSave", func() {
			lastSave := client.LastSave()
			Expect(lastSave.Err()).NotTo(HaveOccurred())
			Expect(lastSave.Val()).NotTo(Equal(0))
		})

		It("should Save", func() {
			// workaround for "ERR Background save already in progress"
			Eventually(func() string {
				return client.Save().Val()
			}, "10s").Should(Equal("OK"))
		})

		It("should SlaveOf", func() {
			slaveOf := client.SlaveOf("localhost", "8888")
			Expect(slaveOf.Err()).NotTo(HaveOccurred())
			Expect(slaveOf.Val()).To(Equal("OK"))

			slaveOf = client.SlaveOf("NO", "ONE")
			Expect(slaveOf.Err()).NotTo(HaveOccurred())
			Expect(slaveOf.Val()).To(Equal("OK"))
		})

		It("should Time", func() {
			time := client.Time()
			Expect(time.Err()).NotTo(HaveOccurred())
			Expect(time.Val()).To(HaveLen(2))
		})

	})

	//------------------------------------------------------------------------------

	Describe("debugging", func() {

		It("should DebugObject", func() {
			debug := client.DebugObject("foo")
			Expect(debug.Err()).To(HaveOccurred())
			Expect(debug.Err().Error()).To(Equal("ERR no such key"))

			client.Set("foo", "bar", 0)
			debug = client.DebugObject("foo")
			Expect(debug.Err()).NotTo(HaveOccurred())
			Expect(debug.Val()).To(ContainSubstring(`serializedlength:4`))
		})

	})

	//------------------------------------------------------------------------------

	Describe("keys", func() {

		It("should Del", func() {
			set := client.Set("key1", "Hello", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))
			set = client.Set("key2", "World", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			del := client.Del("key1", "key2", "key3")
			Expect(del.Err()).NotTo(HaveOccurred())
			Expect(del.Val()).To(Equal(int64(2)))
		})

		It("should Dump", func() {
			set := client.Set("key", "hello", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			dump := client.Dump("key")
			Expect(dump.Err()).NotTo(HaveOccurred())
			Expect(dump.Val()).To(Equal("\x00\x05hello\x06\x00\xf5\x9f\xb7\xf6\x90a\x1c\x99"))
		})

		It("should Exists", func() {
			set := client.Set("key1", "Hello", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			exists := client.Exists("key1")
			Expect(exists.Err()).NotTo(HaveOccurred())
			Expect(exists.Val()).To(Equal(true))

			exists = client.Exists("key2")
			Expect(exists.Err()).NotTo(HaveOccurred())
			Expect(exists.Val()).To(Equal(false))
		})

		It("should Expire", func() {
			set := client.Set("key", "Hello", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			expire := client.Expire("key", 10*time.Second)
			Expect(expire.Err()).NotTo(HaveOccurred())
			Expect(expire.Val()).To(Equal(true))

			ttl := client.TTL("key")
			Expect(ttl.Err()).NotTo(HaveOccurred())
			Expect(ttl.Val()).To(Equal(10 * time.Second))

			set = client.Set("key", "Hello World", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			ttl = client.TTL("key")
			Expect(ttl.Err()).NotTo(HaveOccurred())
			Expect(ttl.Val() < 0).To(Equal(true))
		})

		It("should ExpireAt", func() {
			set := client.Set("key", "Hello", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			exists := client.Exists("key")
			Expect(exists.Err()).NotTo(HaveOccurred())
			Expect(exists.Val()).To(Equal(true))

			expireAt := client.ExpireAt("key", time.Now().Add(-time.Hour))
			Expect(expireAt.Err()).NotTo(HaveOccurred())
			Expect(expireAt.Val()).To(Equal(true))

			exists = client.Exists("key")
			Expect(exists.Err()).NotTo(HaveOccurred())
			Expect(exists.Val()).To(Equal(false))
		})

		It("should Keys", func() {
			mset := client.MSet("one", "1", "two", "2", "three", "3", "four", "4")
			Expect(mset.Err()).NotTo(HaveOccurred())
			Expect(mset.Val()).To(Equal("OK"))

			keys := client.Keys("*o*")
			Expect(keys.Err()).NotTo(HaveOccurred())
			Expect(keys.Val()).To(ConsistOf([]string{"four", "one", "two"}))

			keys = client.Keys("t??")
			Expect(keys.Err()).NotTo(HaveOccurred())
			Expect(keys.Val()).To(Equal([]string{"two"}))

			keys = client.Keys("*")
			Expect(keys.Err()).NotTo(HaveOccurred())
			Expect(keys.Val()).To(ConsistOf([]string{"four", "one", "three", "two"}))
		})

		It("should Migrate", func() {
			migrate := client.Migrate("localhost", redisSecondaryPort, "key", 0, 0)
			Expect(migrate.Err()).NotTo(HaveOccurred())
			Expect(migrate.Val()).To(Equal("NOKEY"))

			set := client.Set("key", "hello", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			migrate = client.Migrate("localhost", redisSecondaryPort, "key", 0, 0)
			Expect(migrate.Err()).To(MatchError("IOERR error or timeout writing to target instance"))
			Expect(migrate.Val()).To(Equal(""))
		})

		It("should Move", func() {
			move := client.Move("key", 1)
			Expect(move.Err()).NotTo(HaveOccurred())
			Expect(move.Val()).To(Equal(false))

			set := client.Set("key", "hello", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			move = client.Move("key", 1)
			Expect(move.Err()).NotTo(HaveOccurred())
			Expect(move.Val()).To(Equal(true))

			get := client.Get("key")
			Expect(get.Err()).To(Equal(redis.Nil))
			Expect(get.Val()).To(Equal(""))

			sel := client.Select(1)
			Expect(sel.Err()).NotTo(HaveOccurred())
			Expect(sel.Val()).To(Equal("OK"))

			get = client.Get("key")
			Expect(get.Err()).NotTo(HaveOccurred())
			Expect(get.Val()).To(Equal("hello"))
			Expect(client.FlushDb().Err()).NotTo(HaveOccurred())
			Expect(client.Select(0).Err()).NotTo(HaveOccurred())
		})

		It("should Object", func() {
			set := client.Set("key", "hello", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			refCount := client.ObjectRefCount("key")
			Expect(refCount.Err()).NotTo(HaveOccurred())
			Expect(refCount.Val()).To(Equal(int64(1)))

			err := client.ObjectEncoding("key").Err()
			Expect(err).NotTo(HaveOccurred())

			idleTime := client.ObjectIdleTime("key")
			Expect(idleTime.Err()).NotTo(HaveOccurred())
			Expect(idleTime.Val()).To(Equal(time.Duration(0)))
		})

		It("should Persist", func() {
			set := client.Set("key", "Hello", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			expire := client.Expire("key", 10*time.Second)
			Expect(expire.Err()).NotTo(HaveOccurred())
			Expect(expire.Val()).To(Equal(true))

			ttl := client.TTL("key")
			Expect(ttl.Err()).NotTo(HaveOccurred())
			Expect(ttl.Val()).To(Equal(10 * time.Second))

			persist := client.Persist("key")
			Expect(persist.Err()).NotTo(HaveOccurred())
			Expect(persist.Val()).To(Equal(true))

			ttl = client.TTL("key")
			Expect(ttl.Err()).NotTo(HaveOccurred())
			Expect(ttl.Val() < 0).To(Equal(true))
		})

		It("should PExpire", func() {
			set := client.Set("key", "Hello", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			expiration := 900 * time.Millisecond
			pexpire := client.PExpire("key", expiration)
			Expect(pexpire.Err()).NotTo(HaveOccurred())
			Expect(pexpire.Val()).To(Equal(true))

			ttl := client.TTL("key")
			Expect(ttl.Err()).NotTo(HaveOccurred())
			Expect(ttl.Val()).To(Equal(time.Second))

			pttl := client.PTTL("key")
			Expect(pttl.Err()).NotTo(HaveOccurred())
			Expect(pttl.Val()).To(BeNumerically("~", expiration, 10*time.Millisecond))
		})

		It("should PExpireAt", func() {
			set := client.Set("key", "Hello", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			expiration := 900 * time.Millisecond
			pexpireat := client.PExpireAt("key", time.Now().Add(expiration))
			Expect(pexpireat.Err()).NotTo(HaveOccurred())
			Expect(pexpireat.Val()).To(Equal(true))

			ttl := client.TTL("key")
			Expect(ttl.Err()).NotTo(HaveOccurred())
			Expect(ttl.Val()).To(Equal(time.Second))

			pttl := client.PTTL("key")
			Expect(pttl.Err()).NotTo(HaveOccurred())
			Expect(pttl.Val()).To(BeNumerically("~", expiration, 10*time.Millisecond))
		})

		It("should PTTL", func() {
			set := client.Set("key", "Hello", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			expiration := time.Second
			expire := client.Expire("key", expiration)
			Expect(expire.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			pttl := client.PTTL("key")
			Expect(pttl.Err()).NotTo(HaveOccurred())
			Expect(pttl.Val()).To(BeNumerically("~", expiration, 10*time.Millisecond))
		})

		It("should RandomKey", func() {
			randomKey := client.RandomKey()
			Expect(randomKey.Err()).To(Equal(redis.Nil))
			Expect(randomKey.Val()).To(Equal(""))

			set := client.Set("key", "hello", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			randomKey = client.RandomKey()
			Expect(randomKey.Err()).NotTo(HaveOccurred())
			Expect(randomKey.Val()).To(Equal("key"))
		})

		It("should Rename", func() {
			set := client.Set("key", "hello", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			status := client.Rename("key", "key1")
			Expect(status.Err()).NotTo(HaveOccurred())
			Expect(status.Val()).To(Equal("OK"))

			get := client.Get("key1")
			Expect(get.Err()).NotTo(HaveOccurred())
			Expect(get.Val()).To(Equal("hello"))
		})

		It("should RenameNX", func() {
			set := client.Set("key", "hello", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			renameNX := client.RenameNX("key", "key1")
			Expect(renameNX.Err()).NotTo(HaveOccurred())
			Expect(renameNX.Val()).To(Equal(true))

			get := client.Get("key1")
			Expect(get.Err()).NotTo(HaveOccurred())
			Expect(get.Val()).To(Equal("hello"))
		})

		It("should Restore", func() {
			err := client.Set("key", "hello", 0).Err()
			Expect(err).NotTo(HaveOccurred())

			dump := client.Dump("key")
			Expect(dump.Err()).NotTo(HaveOccurred())

			err = client.Del("key").Err()
			Expect(err).NotTo(HaveOccurred())

			restore, err := client.Restore("key", 0, dump.Val()).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(restore).To(Equal("OK"))

			type_, err := client.Type("key").Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(type_).To(Equal("string"))

			val, err := client.Get("key").Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal("hello"))
		})

		It("should RestoreReplace", func() {
			err := client.Set("key", "hello", 0).Err()
			Expect(err).NotTo(HaveOccurred())

			dump := client.Dump("key")
			Expect(dump.Err()).NotTo(HaveOccurred())

			restore, err := client.RestoreReplace("key", 0, dump.Val()).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(restore).To(Equal("OK"))

			type_, err := client.Type("key").Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(type_).To(Equal("string"))

			val, err := client.Get("key").Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal("hello"))
		})

		It("should Sort", func() {
			lPush := client.LPush("list", "1")
			Expect(lPush.Err()).NotTo(HaveOccurred())
			Expect(lPush.Val()).To(Equal(int64(1)))
			lPush = client.LPush("list", "3")
			Expect(lPush.Err()).NotTo(HaveOccurred())
			Expect(lPush.Val()).To(Equal(int64(2)))
			lPush = client.LPush("list", "2")
			Expect(lPush.Err()).NotTo(HaveOccurred())
			Expect(lPush.Val()).To(Equal(int64(3)))

			sort := client.Sort("list", redis.Sort{Offset: 0, Count: 2, Order: "ASC"})
			Expect(sort.Err()).NotTo(HaveOccurred())
			Expect(sort.Val()).To(Equal([]string{"1", "2"}))
		})

		It("should TTL", func() {
			ttl := client.TTL("key")
			Expect(ttl.Err()).NotTo(HaveOccurred())
			Expect(ttl.Val() < 0).To(Equal(true))

			set := client.Set("key", "hello", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			expire := client.Expire("key", 60*time.Second)
			Expect(expire.Err()).NotTo(HaveOccurred())
			Expect(expire.Val()).To(Equal(true))

			ttl = client.TTL("key")
			Expect(ttl.Err()).NotTo(HaveOccurred())
			Expect(ttl.Val()).To(Equal(60 * time.Second))
		})

		It("should Type", func() {
			set := client.Set("key", "hello", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			type_ := client.Type("key")
			Expect(type_.Err()).NotTo(HaveOccurred())
			Expect(type_.Val()).To(Equal("string"))
		})

	})

	//------------------------------------------------------------------------------

	Describe("scanning", func() {

		It("should Scan", func() {
			for i := 0; i < 1000; i++ {
				set := client.Set(fmt.Sprintf("key%d", i), "hello", 0)
				Expect(set.Err()).NotTo(HaveOccurred())
			}

			cursor, keys, err := client.Scan(0, "", 0).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(cursor > 0).To(Equal(true))
			Expect(len(keys) > 0).To(Equal(true))
		})

		It("should SScan", func() {
			for i := 0; i < 1000; i++ {
				sadd := client.SAdd("myset", fmt.Sprintf("member%d", i))
				Expect(sadd.Err()).NotTo(HaveOccurred())
			}

			cursor, keys, err := client.SScan("myset", 0, "", 0).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(cursor > 0).To(Equal(true))
			Expect(len(keys) > 0).To(Equal(true))
		})

		It("should HScan", func() {
			for i := 0; i < 1000; i++ {
				sadd := client.HSet("myhash", fmt.Sprintf("key%d", i), "hello")
				Expect(sadd.Err()).NotTo(HaveOccurred())
			}

			cursor, keys, err := client.HScan("myhash", 0, "", 0).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(cursor > 0).To(Equal(true))
			Expect(len(keys) > 0).To(Equal(true))
		})

		It("should ZScan", func() {
			for i := 0; i < 1000; i++ {
				sadd := client.ZAdd("myset", redis.Z{float64(i), fmt.Sprintf("member%d", i)})
				Expect(sadd.Err()).NotTo(HaveOccurred())
			}

			cursor, keys, err := client.ZScan("myset", 0, "", 0).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(cursor > 0).To(Equal(true))
			Expect(len(keys) > 0).To(Equal(true))
		})

	})

	//------------------------------------------------------------------------------

	Describe("strings", func() {

		It("should Append", func() {
			exists := client.Exists("key")
			Expect(exists.Err()).NotTo(HaveOccurred())
			Expect(exists.Val()).To(Equal(false))

			append := client.Append("key", "Hello")
			Expect(append.Err()).NotTo(HaveOccurred())
			Expect(append.Val()).To(Equal(int64(5)))

			append = client.Append("key", " World")
			Expect(append.Err()).NotTo(HaveOccurred())
			Expect(append.Val()).To(Equal(int64(11)))

			get := client.Get("key")
			Expect(get.Err()).NotTo(HaveOccurred())
			Expect(get.Val()).To(Equal("Hello World"))
		})

		It("should BitCount", func() {
			set := client.Set("key", "foobar", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			bitCount := client.BitCount("key", nil)
			Expect(bitCount.Err()).NotTo(HaveOccurred())
			Expect(bitCount.Val()).To(Equal(int64(26)))

			bitCount = client.BitCount("key", &redis.BitCount{0, 0})
			Expect(bitCount.Err()).NotTo(HaveOccurred())
			Expect(bitCount.Val()).To(Equal(int64(4)))

			bitCount = client.BitCount("key", &redis.BitCount{1, 1})
			Expect(bitCount.Err()).NotTo(HaveOccurred())
			Expect(bitCount.Val()).To(Equal(int64(6)))
		})

		It("should BitOpAnd", func() {
			set := client.Set("key1", "1", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			set = client.Set("key2", "0", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			bitOpAnd := client.BitOpAnd("dest", "key1", "key2")
			Expect(bitOpAnd.Err()).NotTo(HaveOccurred())
			Expect(bitOpAnd.Val()).To(Equal(int64(1)))

			get := client.Get("dest")
			Expect(get.Err()).NotTo(HaveOccurred())
			Expect(get.Val()).To(Equal("0"))
		})

		It("should BitOpOr", func() {
			set := client.Set("key1", "1", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			set = client.Set("key2", "0", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			bitOpOr := client.BitOpOr("dest", "key1", "key2")
			Expect(bitOpOr.Err()).NotTo(HaveOccurred())
			Expect(bitOpOr.Val()).To(Equal(int64(1)))

			get := client.Get("dest")
			Expect(get.Err()).NotTo(HaveOccurred())
			Expect(get.Val()).To(Equal("1"))
		})

		It("should BitOpXor", func() {
			set := client.Set("key1", "\xff", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			set = client.Set("key2", "\x0f", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			bitOpXor := client.BitOpXor("dest", "key1", "key2")
			Expect(bitOpXor.Err()).NotTo(HaveOccurred())
			Expect(bitOpXor.Val()).To(Equal(int64(1)))

			get := client.Get("dest")
			Expect(get.Err()).NotTo(HaveOccurred())
			Expect(get.Val()).To(Equal("\xf0"))
		})

		It("should BitOpNot", func() {
			set := client.Set("key1", "\x00", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			bitOpNot := client.BitOpNot("dest", "key1")
			Expect(bitOpNot.Err()).NotTo(HaveOccurred())
			Expect(bitOpNot.Val()).To(Equal(int64(1)))

			get := client.Get("dest")
			Expect(get.Err()).NotTo(HaveOccurred())
			Expect(get.Val()).To(Equal("\xff"))
		})

		It("should BitPos", func() {
			err := client.Set("mykey", "\xff\xf0\x00", 0).Err()
			Expect(err).NotTo(HaveOccurred())

			pos, err := client.BitPos("mykey", 0).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(12)))

			pos, err = client.BitPos("mykey", 1).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(0)))

			pos, err = client.BitPos("mykey", 0, 2).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(16)))

			pos, err = client.BitPos("mykey", 1, 2).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(-1)))

			pos, err = client.BitPos("mykey", 0, -1).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(16)))

			pos, err = client.BitPos("mykey", 1, -1).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(-1)))

			pos, err = client.BitPos("mykey", 0, 2, 1).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(-1)))

			pos, err = client.BitPos("mykey", 0, 0, -3).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(-1)))

			pos, err = client.BitPos("mykey", 0, 0, 0).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(-1)))
		})

		It("should Decr", func() {
			set := client.Set("key", "10", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			decr := client.Decr("key")
			Expect(decr.Err()).NotTo(HaveOccurred())
			Expect(decr.Val()).To(Equal(int64(9)))

			set = client.Set("key", "234293482390480948029348230948", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			decr = client.Decr("key")
			Expect(decr.Err()).To(MatchError("ERR value is not an integer or out of range"))
			Expect(decr.Val()).To(Equal(int64(0)))
		})

		It("should DecrBy", func() {
			set := client.Set("key", "10", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			decrBy := client.DecrBy("key", 5)
			Expect(decrBy.Err()).NotTo(HaveOccurred())
			Expect(decrBy.Val()).To(Equal(int64(5)))
		})

		It("should Get", func() {
			get := client.Get("_")
			Expect(get.Err()).To(Equal(redis.Nil))
			Expect(get.Val()).To(Equal(""))

			set := client.Set("key", "hello", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			get = client.Get("key")
			Expect(get.Err()).NotTo(HaveOccurred())
			Expect(get.Val()).To(Equal("hello"))
		})

		It("should GetBit", func() {
			setBit := client.SetBit("key", 7, 1)
			Expect(setBit.Err()).NotTo(HaveOccurred())
			Expect(setBit.Val()).To(Equal(int64(0)))

			getBit := client.GetBit("key", 0)
			Expect(getBit.Err()).NotTo(HaveOccurred())
			Expect(getBit.Val()).To(Equal(int64(0)))

			getBit = client.GetBit("key", 7)
			Expect(getBit.Err()).NotTo(HaveOccurred())
			Expect(getBit.Val()).To(Equal(int64(1)))

			getBit = client.GetBit("key", 100)
			Expect(getBit.Err()).NotTo(HaveOccurred())
			Expect(getBit.Val()).To(Equal(int64(0)))
		})

		It("should GetRange", func() {
			set := client.Set("key", "This is a string", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			getRange := client.GetRange("key", 0, 3)
			Expect(getRange.Err()).NotTo(HaveOccurred())
			Expect(getRange.Val()).To(Equal("This"))

			getRange = client.GetRange("key", -3, -1)
			Expect(getRange.Err()).NotTo(HaveOccurred())
			Expect(getRange.Val()).To(Equal("ing"))

			getRange = client.GetRange("key", 0, -1)
			Expect(getRange.Err()).NotTo(HaveOccurred())
			Expect(getRange.Val()).To(Equal("This is a string"))

			getRange = client.GetRange("key", 10, 100)
			Expect(getRange.Err()).NotTo(HaveOccurred())
			Expect(getRange.Val()).To(Equal("string"))
		})

		It("should GetSet", func() {
			incr := client.Incr("key")
			Expect(incr.Err()).NotTo(HaveOccurred())
			Expect(incr.Val()).To(Equal(int64(1)))

			getSet := client.GetSet("key", "0")
			Expect(getSet.Err()).NotTo(HaveOccurred())
			Expect(getSet.Val()).To(Equal("1"))

			get := client.Get("key")
			Expect(get.Err()).NotTo(HaveOccurred())
			Expect(get.Val()).To(Equal("0"))
		})

		It("should Incr", func() {
			set := client.Set("key", "10", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			incr := client.Incr("key")
			Expect(incr.Err()).NotTo(HaveOccurred())
			Expect(incr.Val()).To(Equal(int64(11)))

			get := client.Get("key")
			Expect(get.Err()).NotTo(HaveOccurred())
			Expect(get.Val()).To(Equal("11"))
		})

		It("should IncrBy", func() {
			set := client.Set("key", "10", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			incrBy := client.IncrBy("key", 5)
			Expect(incrBy.Err()).NotTo(HaveOccurred())
			Expect(incrBy.Val()).To(Equal(int64(15)))
		})

		It("should IncrByFloat", func() {
			set := client.Set("key", "10.50", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			incrByFloat := client.IncrByFloat("key", 0.1)
			Expect(incrByFloat.Err()).NotTo(HaveOccurred())
			Expect(incrByFloat.Val()).To(Equal(10.6))

			set = client.Set("key", "5.0e3", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			incrByFloat = client.IncrByFloat("key", 2.0e2)
			Expect(incrByFloat.Err()).NotTo(HaveOccurred())
			Expect(incrByFloat.Val()).To(Equal(float64(5200)))
		})

		It("should IncrByFloatOverflow", func() {
			incrByFloat := client.IncrByFloat("key", 996945661)
			Expect(incrByFloat.Err()).NotTo(HaveOccurred())
			Expect(incrByFloat.Val()).To(Equal(float64(996945661)))
		})

		It("should MSetMGet", func() {
			mSet := client.MSet("key1", "hello1", "key2", "hello2")
			Expect(mSet.Err()).NotTo(HaveOccurred())
			Expect(mSet.Val()).To(Equal("OK"))

			mGet := client.MGet("key1", "key2", "_")
			Expect(mGet.Err()).NotTo(HaveOccurred())
			Expect(mGet.Val()).To(Equal([]interface{}{"hello1", "hello2", nil}))
		})

		It("should MSetNX", func() {
			mSetNX := client.MSetNX("key1", "hello1", "key2", "hello2")
			Expect(mSetNX.Err()).NotTo(HaveOccurred())
			Expect(mSetNX.Val()).To(Equal(true))

			mSetNX = client.MSetNX("key2", "hello1", "key3", "hello2")
			Expect(mSetNX.Err()).NotTo(HaveOccurred())
			Expect(mSetNX.Val()).To(Equal(false))
		})

		It("should Set with expiration", func() {
			err := client.Set("key", "hello", 100*time.Millisecond).Err()
			Expect(err).NotTo(HaveOccurred())

			val, err := client.Get("key").Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal("hello"))

			Eventually(func() error {
				return client.Get("foo").Err()
			}, "1s", "100ms").Should(Equal(redis.Nil))
		})

		It("should SetGet", func() {
			set := client.Set("key", "hello", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			get := client.Get("key")
			Expect(get.Err()).NotTo(HaveOccurred())
			Expect(get.Val()).To(Equal("hello"))
		})

		It("should SetNX", func() {
			setNX := client.SetNX("key", "hello", 0)
			Expect(setNX.Err()).NotTo(HaveOccurred())
			Expect(setNX.Val()).To(Equal(true))

			setNX = client.SetNX("key", "hello2", 0)
			Expect(setNX.Err()).NotTo(HaveOccurred())
			Expect(setNX.Val()).To(Equal(false))

			get := client.Get("key")
			Expect(get.Err()).NotTo(HaveOccurred())
			Expect(get.Val()).To(Equal("hello"))
		})

		It("should SetNX with expiration", func() {
			isSet, err := client.SetNX("key", "hello", time.Second).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(isSet).To(Equal(true))

			isSet, err = client.SetNX("key", "hello2", time.Second).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(isSet).To(Equal(false))

			val, err := client.Get("key").Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal("hello"))
		})

		It("should SetXX", func() {
			isSet, err := client.SetXX("key", "hello2", time.Second).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(isSet).To(Equal(false))

			err = client.Set("key", "hello", time.Second).Err()
			Expect(err).NotTo(HaveOccurred())

			isSet, err = client.SetXX("key", "hello2", time.Second).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(isSet).To(Equal(true))

			val, err := client.Get("key").Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal("hello2"))
		})

		It("should SetRange", func() {
			set := client.Set("key", "Hello World", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			range_ := client.SetRange("key", 6, "Redis")
			Expect(range_.Err()).NotTo(HaveOccurred())
			Expect(range_.Val()).To(Equal(int64(11)))

			get := client.Get("key")
			Expect(get.Err()).NotTo(HaveOccurred())
			Expect(get.Val()).To(Equal("Hello Redis"))
		})

		It("should StrLen", func() {
			set := client.Set("key", "hello", 0)
			Expect(set.Err()).NotTo(HaveOccurred())
			Expect(set.Val()).To(Equal("OK"))

			strLen := client.StrLen("key")
			Expect(strLen.Err()).NotTo(HaveOccurred())
			Expect(strLen.Val()).To(Equal(int64(5)))

			strLen = client.StrLen("_")
			Expect(strLen.Err()).NotTo(HaveOccurred())
			Expect(strLen.Val()).To(Equal(int64(0)))
		})

	})

	//------------------------------------------------------------------------------

	Describe("hashes", func() {

		It("should HDel", func() {
			hSet := client.HSet("hash", "key", "hello")
			Expect(hSet.Err()).NotTo(HaveOccurred())

			hDel := client.HDel("hash", "key")
			Expect(hDel.Err()).NotTo(HaveOccurred())
			Expect(hDel.Val()).To(Equal(int64(1)))

			hDel = client.HDel("hash", "key")
			Expect(hDel.Err()).NotTo(HaveOccurred())
			Expect(hDel.Val()).To(Equal(int64(0)))
		})

		It("should HExists", func() {
			hSet := client.HSet("hash", "key", "hello")
			Expect(hSet.Err()).NotTo(HaveOccurred())

			hExists := client.HExists("hash", "key")
			Expect(hExists.Err()).NotTo(HaveOccurred())
			Expect(hExists.Val()).To(Equal(true))

			hExists = client.HExists("hash", "key1")
			Expect(hExists.Err()).NotTo(HaveOccurred())
			Expect(hExists.Val()).To(Equal(false))
		})

		It("should HGet", func() {
			hSet := client.HSet("hash", "key", "hello")
			Expect(hSet.Err()).NotTo(HaveOccurred())

			hGet := client.HGet("hash", "key")
			Expect(hGet.Err()).NotTo(HaveOccurred())
			Expect(hGet.Val()).To(Equal("hello"))

			hGet = client.HGet("hash", "key1")
			Expect(hGet.Err()).To(Equal(redis.Nil))
			Expect(hGet.Val()).To(Equal(""))
		})

		It("should HGetAll", func() {
			hSet := client.HSet("hash", "key1", "hello1")
			Expect(hSet.Err()).NotTo(HaveOccurred())
			hSet = client.HSet("hash", "key2", "hello2")
			Expect(hSet.Err()).NotTo(HaveOccurred())

			hGetAll := client.HGetAll("hash")
			Expect(hGetAll.Err()).NotTo(HaveOccurred())
			Expect(hGetAll.Val()).To(Equal([]string{"key1", "hello1", "key2", "hello2"}))
		})

		It("should HGetAllMap", func() {
			hSet := client.HSet("hash", "key1", "hello1")
			Expect(hSet.Err()).NotTo(HaveOccurred())
			hSet = client.HSet("hash", "key2", "hello2")
			Expect(hSet.Err()).NotTo(HaveOccurred())

			hGetAll := client.HGetAllMap("hash")
			Expect(hGetAll.Err()).NotTo(HaveOccurred())
			Expect(hGetAll.Val()).To(Equal(map[string]string{"key1": "hello1", "key2": "hello2"}))
		})

		It("should HIncrBy", func() {
			hSet := client.HSet("hash", "key", "5")
			Expect(hSet.Err()).NotTo(HaveOccurred())

			hIncrBy := client.HIncrBy("hash", "key", 1)
			Expect(hIncrBy.Err()).NotTo(HaveOccurred())
			Expect(hIncrBy.Val()).To(Equal(int64(6)))

			hIncrBy = client.HIncrBy("hash", "key", -1)
			Expect(hIncrBy.Err()).NotTo(HaveOccurred())
			Expect(hIncrBy.Val()).To(Equal(int64(5)))

			hIncrBy = client.HIncrBy("hash", "key", -10)
			Expect(hIncrBy.Err()).NotTo(HaveOccurred())
			Expect(hIncrBy.Val()).To(Equal(int64(-5)))
		})

		It("should HIncrByFloat", func() {
			hSet := client.HSet("hash", "field", "10.50")
			Expect(hSet.Err()).NotTo(HaveOccurred())
			Expect(hSet.Val()).To(Equal(true))

			hIncrByFloat := client.HIncrByFloat("hash", "field", 0.1)
			Expect(hIncrByFloat.Err()).NotTo(HaveOccurred())
			Expect(hIncrByFloat.Val()).To(Equal(10.6))

			hSet = client.HSet("hash", "field", "5.0e3")
			Expect(hSet.Err()).NotTo(HaveOccurred())
			Expect(hSet.Val()).To(Equal(false))

			hIncrByFloat = client.HIncrByFloat("hash", "field", 2.0e2)
			Expect(hIncrByFloat.Err()).NotTo(HaveOccurred())
			Expect(hIncrByFloat.Val()).To(Equal(float64(5200)))
		})

		It("should HKeys", func() {
			hkeys := client.HKeys("hash")
			Expect(hkeys.Err()).NotTo(HaveOccurred())
			Expect(hkeys.Val()).To(Equal([]string{}))

			hset := client.HSet("hash", "key1", "hello1")
			Expect(hset.Err()).NotTo(HaveOccurred())
			hset = client.HSet("hash", "key2", "hello2")
			Expect(hset.Err()).NotTo(HaveOccurred())

			hkeys = client.HKeys("hash")
			Expect(hkeys.Err()).NotTo(HaveOccurred())
			Expect(hkeys.Val()).To(Equal([]string{"key1", "key2"}))
		})

		It("should HLen", func() {
			hSet := client.HSet("hash", "key1", "hello1")
			Expect(hSet.Err()).NotTo(HaveOccurred())
			hSet = client.HSet("hash", "key2", "hello2")
			Expect(hSet.Err()).NotTo(HaveOccurred())

			hLen := client.HLen("hash")
			Expect(hLen.Err()).NotTo(HaveOccurred())
			Expect(hLen.Val()).To(Equal(int64(2)))
		})

		It("should HMGet", func() {
			hSet := client.HSet("hash", "key1", "hello1")
			Expect(hSet.Err()).NotTo(HaveOccurred())
			hSet = client.HSet("hash", "key2", "hello2")
			Expect(hSet.Err()).NotTo(HaveOccurred())

			hMGet := client.HMGet("hash", "key1", "key2", "_")
			Expect(hMGet.Err()).NotTo(HaveOccurred())
			Expect(hMGet.Val()).To(Equal([]interface{}{"hello1", "hello2", nil}))
		})

		It("should HMSet", func() {
			hMSet := client.HMSet("hash", "key1", "hello1", "key2", "hello2")
			Expect(hMSet.Err()).NotTo(HaveOccurred())
			Expect(hMSet.Val()).To(Equal("OK"))

			hGet := client.HGet("hash", "key1")
			Expect(hGet.Err()).NotTo(HaveOccurred())
			Expect(hGet.Val()).To(Equal("hello1"))

			hGet = client.HGet("hash", "key2")
			Expect(hGet.Err()).NotTo(HaveOccurred())
			Expect(hGet.Val()).To(Equal("hello2"))
		})

		It("should HSet", func() {
			hSet := client.HSet("hash", "key", "hello")
			Expect(hSet.Err()).NotTo(HaveOccurred())
			Expect(hSet.Val()).To(Equal(true))

			hGet := client.HGet("hash", "key")
			Expect(hGet.Err()).NotTo(HaveOccurred())
			Expect(hGet.Val()).To(Equal("hello"))
		})

		It("should HSetNX", func() {
			hSetNX := client.HSetNX("hash", "key", "hello")
			Expect(hSetNX.Err()).NotTo(HaveOccurred())
			Expect(hSetNX.Val()).To(Equal(true))

			hSetNX = client.HSetNX("hash", "key", "hello")
			Expect(hSetNX.Err()).NotTo(HaveOccurred())
			Expect(hSetNX.Val()).To(Equal(false))

			hGet := client.HGet("hash", "key")
			Expect(hGet.Err()).NotTo(HaveOccurred())
			Expect(hGet.Val()).To(Equal("hello"))
		})

		It("should HVals", func() {
			hSet := client.HSet("hash", "key1", "hello1")
			Expect(hSet.Err()).NotTo(HaveOccurred())
			hSet = client.HSet("hash", "key2", "hello2")
			Expect(hSet.Err()).NotTo(HaveOccurred())

			hVals := client.HVals("hash")
			Expect(hVals.Err()).NotTo(HaveOccurred())
			Expect(hVals.Val()).To(Equal([]string{"hello1", "hello2"}))
		})

	})

	//------------------------------------------------------------------------------

	Describe("lists", func() {

		It("should BLPop", func() {
			rPush := client.RPush("list1", "a", "b", "c")
			Expect(rPush.Err()).NotTo(HaveOccurred())

			bLPop := client.BLPop(0, "list1", "list2")
			Expect(bLPop.Err()).NotTo(HaveOccurred())
			Expect(bLPop.Val()).To(Equal([]string{"list1", "a"}))
		})

		It("should BLPopBlocks", func() {
			started := make(chan bool)
			done := make(chan bool)
			go func() {
				defer GinkgoRecover()

				started <- true
				bLPop := client.BLPop(0, "list")
				Expect(bLPop.Err()).NotTo(HaveOccurred())
				Expect(bLPop.Val()).To(Equal([]string{"list", "a"}))
				done <- true
			}()
			<-started

			select {
			case <-done:
				Fail("BLPop is not blocked")
			case <-time.After(time.Second):
				// ok
			}

			rPush := client.RPush("list", "a")
			Expect(rPush.Err()).NotTo(HaveOccurred())

			select {
			case <-done:
				// ok
			case <-time.After(time.Second):
				Fail("BLPop is still blocked")
			}
		})

		It("should BLPop timeout", func() {
			bLPop := client.BLPop(time.Second, "list1")
			Expect(bLPop.Val()).To(BeNil())
			Expect(bLPop.Err()).To(Equal(redis.Nil))
		})

		It("should BRPop", func() {
			rPush := client.RPush("list1", "a", "b", "c")
			Expect(rPush.Err()).NotTo(HaveOccurred())

			bRPop := client.BRPop(0, "list1", "list2")
			Expect(bRPop.Err()).NotTo(HaveOccurred())
			Expect(bRPop.Val()).To(Equal([]string{"list1", "c"}))
		})

		It("should BRPop blocks", func() {
			started := make(chan bool)
			done := make(chan bool)
			go func() {
				defer GinkgoRecover()

				started <- true
				brpop := client.BRPop(0, "list")
				Expect(brpop.Err()).NotTo(HaveOccurred())
				Expect(brpop.Val()).To(Equal([]string{"list", "a"}))
				done <- true
			}()
			<-started

			select {
			case <-done:
				Fail("BRPop is not blocked")
			case <-time.After(time.Second):
				// ok
			}

			rPush := client.RPush("list", "a")
			Expect(rPush.Err()).NotTo(HaveOccurred())

			select {
			case <-done:
				// ok
			case <-time.After(time.Second):
				Fail("BRPop is still blocked")
				// ok
			}
		})

		It("should BRPopLPush", func() {
			rPush := client.RPush("list1", "a", "b", "c")
			Expect(rPush.Err()).NotTo(HaveOccurred())

			bRPopLPush := client.BRPopLPush("list1", "list2", 0)
			Expect(bRPopLPush.Err()).NotTo(HaveOccurred())
			Expect(bRPopLPush.Val()).To(Equal("c"))
		})

		It("should LIndex", func() {
			lPush := client.LPush("list", "World")
			Expect(lPush.Err()).NotTo(HaveOccurred())
			lPush = client.LPush("list", "Hello")
			Expect(lPush.Err()).NotTo(HaveOccurred())

			lIndex := client.LIndex("list", 0)
			Expect(lIndex.Err()).NotTo(HaveOccurred())
			Expect(lIndex.Val()).To(Equal("Hello"))

			lIndex = client.LIndex("list", -1)
			Expect(lIndex.Err()).NotTo(HaveOccurred())
			Expect(lIndex.Val()).To(Equal("World"))

			lIndex = client.LIndex("list", 3)
			Expect(lIndex.Err()).To(Equal(redis.Nil))
			Expect(lIndex.Val()).To(Equal(""))
		})

		It("should LInsert", func() {
			rPush := client.RPush("list", "Hello")
			Expect(rPush.Err()).NotTo(HaveOccurred())
			rPush = client.RPush("list", "World")
			Expect(rPush.Err()).NotTo(HaveOccurred())

			lInsert := client.LInsert("list", "BEFORE", "World", "There")
			Expect(lInsert.Err()).NotTo(HaveOccurred())
			Expect(lInsert.Val()).To(Equal(int64(3)))

			lRange := client.LRange("list", 0, -1)
			Expect(lRange.Err()).NotTo(HaveOccurred())
			Expect(lRange.Val()).To(Equal([]string{"Hello", "There", "World"}))
		})

		It("should LLen", func() {
			lPush := client.LPush("list", "World")
			Expect(lPush.Err()).NotTo(HaveOccurred())
			lPush = client.LPush("list", "Hello")
			Expect(lPush.Err()).NotTo(HaveOccurred())

			lLen := client.LLen("list")
			Expect(lLen.Err()).NotTo(HaveOccurred())
			Expect(lLen.Val()).To(Equal(int64(2)))
		})

		It("should LPop", func() {
			rPush := client.RPush("list", "one")
			Expect(rPush.Err()).NotTo(HaveOccurred())
			rPush = client.RPush("list", "two")
			Expect(rPush.Err()).NotTo(HaveOccurred())
			rPush = client.RPush("list", "three")
			Expect(rPush.Err()).NotTo(HaveOccurred())

			lPop := client.LPop("list")
			Expect(lPop.Err()).NotTo(HaveOccurred())
			Expect(lPop.Val()).To(Equal("one"))

			lRange := client.LRange("list", 0, -1)
			Expect(lRange.Err()).NotTo(HaveOccurred())
			Expect(lRange.Val()).To(Equal([]string{"two", "three"}))
		})

		It("should LPush", func() {
			lPush := client.LPush("list", "World")
			Expect(lPush.Err()).NotTo(HaveOccurred())
			lPush = client.LPush("list", "Hello")
			Expect(lPush.Err()).NotTo(HaveOccurred())

			lRange := client.LRange("list", 0, -1)
			Expect(lRange.Err()).NotTo(HaveOccurred())
			Expect(lRange.Val()).To(Equal([]string{"Hello", "World"}))
		})

		It("should LPushX", func() {
			lPush := client.LPush("list", "World")
			Expect(lPush.Err()).NotTo(HaveOccurred())

			lPushX := client.LPushX("list", "Hello")
			Expect(lPushX.Err()).NotTo(HaveOccurred())
			Expect(lPushX.Val()).To(Equal(int64(2)))

			lPushX = client.LPushX("list2", "Hello")
			Expect(lPushX.Err()).NotTo(HaveOccurred())
			Expect(lPushX.Val()).To(Equal(int64(0)))

			lRange := client.LRange("list", 0, -1)
			Expect(lRange.Err()).NotTo(HaveOccurred())
			Expect(lRange.Val()).To(Equal([]string{"Hello", "World"}))

			lRange = client.LRange("list2", 0, -1)
			Expect(lRange.Err()).NotTo(HaveOccurred())
			Expect(lRange.Val()).To(Equal([]string{}))
		})

		It("should LRange", func() {
			rPush := client.RPush("list", "one")
			Expect(rPush.Err()).NotTo(HaveOccurred())
			rPush = client.RPush("list", "two")
			Expect(rPush.Err()).NotTo(HaveOccurred())
			rPush = client.RPush("list", "three")
			Expect(rPush.Err()).NotTo(HaveOccurred())

			lRange := client.LRange("list", 0, 0)
			Expect(lRange.Err()).NotTo(HaveOccurred())
			Expect(lRange.Val()).To(Equal([]string{"one"}))

			lRange = client.LRange("list", -3, 2)
			Expect(lRange.Err()).NotTo(HaveOccurred())
			Expect(lRange.Val()).To(Equal([]string{"one", "two", "three"}))

			lRange = client.LRange("list", -100, 100)
			Expect(lRange.Err()).NotTo(HaveOccurred())
			Expect(lRange.Val()).To(Equal([]string{"one", "two", "three"}))

			lRange = client.LRange("list", 5, 10)
			Expect(lRange.Err()).NotTo(HaveOccurred())
			Expect(lRange.Val()).To(Equal([]string{}))
		})

		It("should LRem", func() {
			rPush := client.RPush("list", "hello")
			Expect(rPush.Err()).NotTo(HaveOccurred())
			rPush = client.RPush("list", "hello")
			Expect(rPush.Err()).NotTo(HaveOccurred())
			rPush = client.RPush("list", "key")
			Expect(rPush.Err()).NotTo(HaveOccurred())
			rPush = client.RPush("list", "hello")
			Expect(rPush.Err()).NotTo(HaveOccurred())

			lRem := client.LRem("list", -2, "hello")
			Expect(lRem.Err()).NotTo(HaveOccurred())
			Expect(lRem.Val()).To(Equal(int64(2)))

			lRange := client.LRange("list", 0, -1)
			Expect(lRange.Err()).NotTo(HaveOccurred())
			Expect(lRange.Val()).To(Equal([]string{"hello", "key"}))
		})

		It("should LSet", func() {
			rPush := client.RPush("list", "one")
			Expect(rPush.Err()).NotTo(HaveOccurred())
			rPush = client.RPush("list", "two")
			Expect(rPush.Err()).NotTo(HaveOccurred())
			rPush = client.RPush("list", "three")
			Expect(rPush.Err()).NotTo(HaveOccurred())

			lSet := client.LSet("list", 0, "four")
			Expect(lSet.Err()).NotTo(HaveOccurred())
			Expect(lSet.Val()).To(Equal("OK"))

			lSet = client.LSet("list", -2, "five")
			Expect(lSet.Err()).NotTo(HaveOccurred())
			Expect(lSet.Val()).To(Equal("OK"))

			lRange := client.LRange("list", 0, -1)
			Expect(lRange.Err()).NotTo(HaveOccurred())
			Expect(lRange.Val()).To(Equal([]string{"four", "five", "three"}))
		})

		It("should LTrim", func() {
			rPush := client.RPush("list", "one")
			Expect(rPush.Err()).NotTo(HaveOccurred())
			rPush = client.RPush("list", "two")
			Expect(rPush.Err()).NotTo(HaveOccurred())
			rPush = client.RPush("list", "three")
			Expect(rPush.Err()).NotTo(HaveOccurred())

			lTrim := client.LTrim("list", 1, -1)
			Expect(lTrim.Err()).NotTo(HaveOccurred())
			Expect(lTrim.Val()).To(Equal("OK"))

			lRange := client.LRange("list", 0, -1)
			Expect(lRange.Err()).NotTo(HaveOccurred())
			Expect(lRange.Val()).To(Equal([]string{"two", "three"}))
		})

		It("should RPop", func() {
			rPush := client.RPush("list", "one")
			Expect(rPush.Err()).NotTo(HaveOccurred())
			rPush = client.RPush("list", "two")
			Expect(rPush.Err()).NotTo(HaveOccurred())
			rPush = client.RPush("list", "three")
			Expect(rPush.Err()).NotTo(HaveOccurred())

			rPop := client.RPop("list")
			Expect(rPop.Err()).NotTo(HaveOccurred())
			Expect(rPop.Val()).To(Equal("three"))

			lRange := client.LRange("list", 0, -1)
			Expect(lRange.Err()).NotTo(HaveOccurred())
			Expect(lRange.Val()).To(Equal([]string{"one", "two"}))
		})

		It("should RPopLPush", func() {
			rPush := client.RPush("list", "one")
			Expect(rPush.Err()).NotTo(HaveOccurred())
			rPush = client.RPush("list", "two")
			Expect(rPush.Err()).NotTo(HaveOccurred())
			rPush = client.RPush("list", "three")
			Expect(rPush.Err()).NotTo(HaveOccurred())

			rPopLPush := client.RPopLPush("list", "list2")
			Expect(rPopLPush.Err()).NotTo(HaveOccurred())
			Expect(rPopLPush.Val()).To(Equal("three"))

			lRange := client.LRange("list", 0, -1)
			Expect(lRange.Err()).NotTo(HaveOccurred())
			Expect(lRange.Val()).To(Equal([]string{"one", "two"}))

			lRange = client.LRange("list2", 0, -1)
			Expect(lRange.Err()).NotTo(HaveOccurred())
			Expect(lRange.Val()).To(Equal([]string{"three"}))
		})

		It("should RPush", func() {
			rPush := client.RPush("list", "Hello")
			Expect(rPush.Err()).NotTo(HaveOccurred())
			Expect(rPush.Val()).To(Equal(int64(1)))

			rPush = client.RPush("list", "World")
			Expect(rPush.Err()).NotTo(HaveOccurred())
			Expect(rPush.Val()).To(Equal(int64(2)))

			lRange := client.LRange("list", 0, -1)
			Expect(lRange.Err()).NotTo(HaveOccurred())
			Expect(lRange.Val()).To(Equal([]string{"Hello", "World"}))
		})

		It("should RPushX", func() {
			rPush := client.RPush("list", "Hello")
			Expect(rPush.Err()).NotTo(HaveOccurred())
			Expect(rPush.Val()).To(Equal(int64(1)))

			rPushX := client.RPushX("list", "World")
			Expect(rPushX.Err()).NotTo(HaveOccurred())
			Expect(rPushX.Val()).To(Equal(int64(2)))

			rPushX = client.RPushX("list2", "World")
			Expect(rPushX.Err()).NotTo(HaveOccurred())
			Expect(rPushX.Val()).To(Equal(int64(0)))

			lRange := client.LRange("list", 0, -1)
			Expect(lRange.Err()).NotTo(HaveOccurred())
			Expect(lRange.Val()).To(Equal([]string{"Hello", "World"}))

			lRange = client.LRange("list2", 0, -1)
			Expect(lRange.Err()).NotTo(HaveOccurred())
			Expect(lRange.Val()).To(Equal([]string{}))
		})

	})

	//------------------------------------------------------------------------------

	Describe("sets", func() {

		It("should SAdd", func() {
			sAdd := client.SAdd("set", "Hello")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			Expect(sAdd.Val()).To(Equal(int64(1)))

			sAdd = client.SAdd("set", "World")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			Expect(sAdd.Val()).To(Equal(int64(1)))

			sAdd = client.SAdd("set", "World")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			Expect(sAdd.Val()).To(Equal(int64(0)))

			sMembers := client.SMembers("set")
			Expect(sMembers.Err()).NotTo(HaveOccurred())
			Expect(sMembers.Val()).To(ConsistOf([]string{"Hello", "World"}))
		})

		It("should SCard", func() {
			sAdd := client.SAdd("set", "Hello")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			Expect(sAdd.Val()).To(Equal(int64(1)))

			sAdd = client.SAdd("set", "World")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			Expect(sAdd.Val()).To(Equal(int64(1)))

			sCard := client.SCard("set")
			Expect(sCard.Err()).NotTo(HaveOccurred())
			Expect(sCard.Val()).To(Equal(int64(2)))
		})

		It("should SDiff", func() {
			sAdd := client.SAdd("set1", "a")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set1", "b")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set1", "c")
			Expect(sAdd.Err()).NotTo(HaveOccurred())

			sAdd = client.SAdd("set2", "c")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set2", "d")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set2", "e")
			Expect(sAdd.Err()).NotTo(HaveOccurred())

			sDiff := client.SDiff("set1", "set2")
			Expect(sDiff.Err()).NotTo(HaveOccurred())
			Expect(sDiff.Val()).To(ConsistOf([]string{"a", "b"}))
		})

		It("should SDiffStore", func() {
			sAdd := client.SAdd("set1", "a")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set1", "b")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set1", "c")
			Expect(sAdd.Err()).NotTo(HaveOccurred())

			sAdd = client.SAdd("set2", "c")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set2", "d")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set2", "e")
			Expect(sAdd.Err()).NotTo(HaveOccurred())

			sDiffStore := client.SDiffStore("set", "set1", "set2")
			Expect(sDiffStore.Err()).NotTo(HaveOccurred())
			Expect(sDiffStore.Val()).To(Equal(int64(2)))

			sMembers := client.SMembers("set")
			Expect(sMembers.Err()).NotTo(HaveOccurred())
			Expect(sMembers.Val()).To(ConsistOf([]string{"a", "b"}))
		})

		It("should SInter", func() {
			sAdd := client.SAdd("set1", "a")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set1", "b")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set1", "c")
			Expect(sAdd.Err()).NotTo(HaveOccurred())

			sAdd = client.SAdd("set2", "c")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set2", "d")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set2", "e")
			Expect(sAdd.Err()).NotTo(HaveOccurred())

			sInter := client.SInter("set1", "set2")
			Expect(sInter.Err()).NotTo(HaveOccurred())
			Expect(sInter.Val()).To(Equal([]string{"c"}))
		})

		It("should SInterStore", func() {
			sAdd := client.SAdd("set1", "a")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set1", "b")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set1", "c")
			Expect(sAdd.Err()).NotTo(HaveOccurred())

			sAdd = client.SAdd("set2", "c")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set2", "d")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set2", "e")
			Expect(sAdd.Err()).NotTo(HaveOccurred())

			sInterStore := client.SInterStore("set", "set1", "set2")
			Expect(sInterStore.Err()).NotTo(HaveOccurred())
			Expect(sInterStore.Val()).To(Equal(int64(1)))

			sMembers := client.SMembers("set")
			Expect(sMembers.Err()).NotTo(HaveOccurred())
			Expect(sMembers.Val()).To(Equal([]string{"c"}))
		})

		It("should IsMember", func() {
			sAdd := client.SAdd("set", "one")
			Expect(sAdd.Err()).NotTo(HaveOccurred())

			sIsMember := client.SIsMember("set", "one")
			Expect(sIsMember.Err()).NotTo(HaveOccurred())
			Expect(sIsMember.Val()).To(Equal(true))

			sIsMember = client.SIsMember("set", "two")
			Expect(sIsMember.Err()).NotTo(HaveOccurred())
			Expect(sIsMember.Val()).To(Equal(false))
		})

		It("should SMembers", func() {
			sAdd := client.SAdd("set", "Hello")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set", "World")
			Expect(sAdd.Err()).NotTo(HaveOccurred())

			sMembers := client.SMembers("set")
			Expect(sMembers.Err()).NotTo(HaveOccurred())
			Expect(sMembers.Val()).To(ConsistOf([]string{"Hello", "World"}))
		})

		It("should SMove", func() {
			sAdd := client.SAdd("set1", "one")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set1", "two")
			Expect(sAdd.Err()).NotTo(HaveOccurred())

			sAdd = client.SAdd("set2", "three")
			Expect(sAdd.Err()).NotTo(HaveOccurred())

			sMove := client.SMove("set1", "set2", "two")
			Expect(sMove.Err()).NotTo(HaveOccurred())
			Expect(sMove.Val()).To(Equal(true))

			sMembers := client.SMembers("set1")
			Expect(sMembers.Err()).NotTo(HaveOccurred())
			Expect(sMembers.Val()).To(Equal([]string{"one"}))

			sMembers = client.SMembers("set2")
			Expect(sMembers.Err()).NotTo(HaveOccurred())
			Expect(sMembers.Val()).To(ConsistOf([]string{"three", "two"}))
		})

		It("should SPop", func() {
			sAdd := client.SAdd("set", "one")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set", "two")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set", "three")
			Expect(sAdd.Err()).NotTo(HaveOccurred())

			sPop := client.SPop("set")
			Expect(sPop.Err()).NotTo(HaveOccurred())
			Expect(sPop.Val()).NotTo(Equal(""))

			sMembers := client.SMembers("set")
			Expect(sMembers.Err()).NotTo(HaveOccurred())
			Expect(sMembers.Val()).To(HaveLen(2))
		})

		It("should SRandMember", func() {
			sAdd := client.SAdd("set", "one")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set", "two")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set", "three")
			Expect(sAdd.Err()).NotTo(HaveOccurred())

			sRandMember := client.SRandMember("set")
			Expect(sRandMember.Err()).NotTo(HaveOccurred())
			Expect(sRandMember.Val()).NotTo(Equal(""))

			sMembers := client.SMembers("set")
			Expect(sMembers.Err()).NotTo(HaveOccurred())
			Expect(sMembers.Val()).To(HaveLen(3))
		})

		It("should SRem", func() {
			sAdd := client.SAdd("set", "one")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set", "two")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set", "three")
			Expect(sAdd.Err()).NotTo(HaveOccurred())

			sRem := client.SRem("set", "one")
			Expect(sRem.Err()).NotTo(HaveOccurred())
			Expect(sRem.Val()).To(Equal(int64(1)))

			sRem = client.SRem("set", "four")
			Expect(sRem.Err()).NotTo(HaveOccurred())
			Expect(sRem.Val()).To(Equal(int64(0)))

			sMembers := client.SMembers("set")
			Expect(sMembers.Err()).NotTo(HaveOccurred())
			Expect(sMembers.Val()).To(ConsistOf([]string{"three", "two"}))
		})

		It("should SUnion", func() {
			sAdd := client.SAdd("set1", "a")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set1", "b")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set1", "c")
			Expect(sAdd.Err()).NotTo(HaveOccurred())

			sAdd = client.SAdd("set2", "c")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set2", "d")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set2", "e")
			Expect(sAdd.Err()).NotTo(HaveOccurred())

			sUnion := client.SUnion("set1", "set2")
			Expect(sUnion.Err()).NotTo(HaveOccurred())
			Expect(sUnion.Val()).To(HaveLen(5))
		})

		It("should SUnionStore", func() {
			sAdd := client.SAdd("set1", "a")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set1", "b")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set1", "c")
			Expect(sAdd.Err()).NotTo(HaveOccurred())

			sAdd = client.SAdd("set2", "c")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set2", "d")
			Expect(sAdd.Err()).NotTo(HaveOccurred())
			sAdd = client.SAdd("set2", "e")
			Expect(sAdd.Err()).NotTo(HaveOccurred())

			sUnionStore := client.SUnionStore("set", "set1", "set2")
			Expect(sUnionStore.Err()).NotTo(HaveOccurred())
			Expect(sUnionStore.Val()).To(Equal(int64(5)))

			sMembers := client.SMembers("set")
			Expect(sMembers.Err()).NotTo(HaveOccurred())
			Expect(sMembers.Val()).To(HaveLen(5))
		})

	})

	//------------------------------------------------------------------------------

	Describe("sorted sets", func() {

		It("should ZAdd", func() {
			added, err := client.ZAdd("zset", redis.Z{1, "one"}).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(added).To(Equal(int64(1)))

			added, err = client.ZAdd("zset", redis.Z{1, "uno"}).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(added).To(Equal(int64(1)))

			added, err = client.ZAdd("zset", redis.Z{2, "two"}).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(added).To(Equal(int64(1)))

			added, err = client.ZAdd("zset", redis.Z{3, "two"}).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(added).To(Equal(int64(0)))

			val, err := client.ZRangeWithScores("zset", 0, -1).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal([]redis.Z{{1, "one"}, {1, "uno"}, {3, "two"}}))
		})

		It("should ZAdd bytes", func() {
			added, err := client.ZAdd("zset", redis.Z{1, []byte("one")}).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(added).To(Equal(int64(1)))

			added, err = client.ZAdd("zset", redis.Z{1, []byte("uno")}).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(added).To(Equal(int64(1)))

			added, err = client.ZAdd("zset", redis.Z{2, []byte("two")}).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(added).To(Equal(int64(1)))

			added, err = client.ZAdd("zset", redis.Z{3, []byte("two")}).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(added).To(Equal(int64(0)))

			val, err := client.ZRangeWithScores("zset", 0, -1).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal([]redis.Z{{1, "one"}, {1, "uno"}, {3, "two"}}))
		})

		It("should ZCard", func() {
			zAdd := client.ZAdd("zset", redis.Z{1, "one"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{2, "two"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())

			zCard := client.ZCard("zset")
			Expect(zCard.Err()).NotTo(HaveOccurred())
			Expect(zCard.Val()).To(Equal(int64(2)))
		})

		It("should ZCount", func() {
			zAdd := client.ZAdd("zset", redis.Z{1, "one"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{2, "two"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{3, "three"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())

			zCount := client.ZCount("zset", "-inf", "+inf")
			Expect(zCount.Err()).NotTo(HaveOccurred())
			Expect(zCount.Val()).To(Equal(int64(3)))

			zCount = client.ZCount("zset", "(1", "3")
			Expect(zCount.Err()).NotTo(HaveOccurred())
			Expect(zCount.Val()).To(Equal(int64(2)))
		})

		It("should ZIncrBy", func() {
			zAdd := client.ZAdd("zset", redis.Z{1, "one"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{2, "two"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())

			zIncrBy := client.ZIncrBy("zset", 2, "one")
			Expect(zIncrBy.Err()).NotTo(HaveOccurred())
			Expect(zIncrBy.Val()).To(Equal(float64(3)))

			val, err := client.ZRangeWithScores("zset", 0, -1).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal([]redis.Z{{2, "two"}, {3, "one"}}))
		})

		It("should ZInterStore", func() {
			zAdd := client.ZAdd("zset1", redis.Z{1, "one"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset1", redis.Z{2, "two"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())

			zAdd = client.ZAdd("zset2", redis.Z{1, "one"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset2", redis.Z{2, "two"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset3", redis.Z{3, "two"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())

			zInterStore := client.ZInterStore(
				"out", redis.ZStore{Weights: []int64{2, 3}}, "zset1", "zset2")
			Expect(zInterStore.Err()).NotTo(HaveOccurred())
			Expect(zInterStore.Val()).To(Equal(int64(2)))

			val, err := client.ZRangeWithScores("out", 0, -1).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal([]redis.Z{{5, "one"}, {10, "two"}}))
		})

		It("should ZRange", func() {
			zAdd := client.ZAdd("zset", redis.Z{1, "one"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{2, "two"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{3, "three"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())

			zRange := client.ZRange("zset", 0, -1)
			Expect(zRange.Err()).NotTo(HaveOccurred())
			Expect(zRange.Val()).To(Equal([]string{"one", "two", "three"}))

			zRange = client.ZRange("zset", 2, 3)
			Expect(zRange.Err()).NotTo(HaveOccurred())
			Expect(zRange.Val()).To(Equal([]string{"three"}))

			zRange = client.ZRange("zset", -2, -1)
			Expect(zRange.Err()).NotTo(HaveOccurred())
			Expect(zRange.Val()).To(Equal([]string{"two", "three"}))
		})

		It("should ZRangeWithScores", func() {
			zAdd := client.ZAdd("zset", redis.Z{1, "one"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{2, "two"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{3, "three"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())

			val, err := client.ZRangeWithScores("zset", 0, -1).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal([]redis.Z{{1, "one"}, {2, "two"}, {3, "three"}}))

			val, err = client.ZRangeWithScores("zset", 2, 3).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal([]redis.Z{{3, "three"}}))

			val, err = client.ZRangeWithScores("zset", -2, -1).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal([]redis.Z{{2, "two"}, {3, "three"}}))
		})

		It("should ZRangeByScore", func() {
			zAdd := client.ZAdd("zset", redis.Z{1, "one"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{2, "two"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{3, "three"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())

			zRangeByScore := client.ZRangeByScore("zset", redis.ZRangeByScore{
				Min: "-inf",
				Max: "+inf",
			})
			Expect(zRangeByScore.Err()).NotTo(HaveOccurred())
			Expect(zRangeByScore.Val()).To(Equal([]string{"one", "two", "three"}))

			zRangeByScore = client.ZRangeByScore("zset", redis.ZRangeByScore{
				Min: "1",
				Max: "2",
			})
			Expect(zRangeByScore.Err()).NotTo(HaveOccurred())
			Expect(zRangeByScore.Val()).To(Equal([]string{"one", "two"}))

			zRangeByScore = client.ZRangeByScore("zset", redis.ZRangeByScore{
				Min: "(1",
				Max: "2",
			})
			Expect(zRangeByScore.Err()).NotTo(HaveOccurred())
			Expect(zRangeByScore.Val()).To(Equal([]string{"two"}))

			zRangeByScore = client.ZRangeByScore("zset", redis.ZRangeByScore{
				Min: "(1",
				Max: "(2",
			})
			Expect(zRangeByScore.Err()).NotTo(HaveOccurred())
			Expect(zRangeByScore.Val()).To(Equal([]string{}))
		})

		It("should ZRangeByScoreWithScoresMap", func() {
			zAdd := client.ZAdd("zset", redis.Z{1, "one"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{2, "two"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{3, "three"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())

			val, err := client.ZRangeByScoreWithScores("zset", redis.ZRangeByScore{
				Min: "-inf",
				Max: "+inf",
			}).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal([]redis.Z{{1, "one"}, {2, "two"}, {3, "three"}}))

			val, err = client.ZRangeByScoreWithScores("zset", redis.ZRangeByScore{
				Min: "1",
				Max: "2",
			}).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal([]redis.Z{{1, "one"}, {2, "two"}}))

			val, err = client.ZRangeByScoreWithScores("zset", redis.ZRangeByScore{
				Min: "(1",
				Max: "2",
			}).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal([]redis.Z{{2, "two"}}))

			val, err = client.ZRangeByScoreWithScores("zset", redis.ZRangeByScore{
				Min: "(1",
				Max: "(2",
			}).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal([]redis.Z{}))
		})

		It("should ZRank", func() {
			zAdd := client.ZAdd("zset", redis.Z{1, "one"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{2, "two"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{3, "three"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())

			zRank := client.ZRank("zset", "three")
			Expect(zRank.Err()).NotTo(HaveOccurred())
			Expect(zRank.Val()).To(Equal(int64(2)))

			zRank = client.ZRank("zset", "four")
			Expect(zRank.Err()).To(Equal(redis.Nil))
			Expect(zRank.Val()).To(Equal(int64(0)))
		})

		It("should ZRem", func() {
			zAdd := client.ZAdd("zset", redis.Z{1, "one"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{2, "two"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{3, "three"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())

			zRem := client.ZRem("zset", "two")
			Expect(zRem.Err()).NotTo(HaveOccurred())
			Expect(zRem.Val()).To(Equal(int64(1)))

			val, err := client.ZRangeWithScores("zset", 0, -1).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal([]redis.Z{{1, "one"}, {3, "three"}}))
		})

		It("should ZRemRangeByRank", func() {
			zAdd := client.ZAdd("zset", redis.Z{1, "one"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{2, "two"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{3, "three"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())

			zRemRangeByRank := client.ZRemRangeByRank("zset", 0, 1)
			Expect(zRemRangeByRank.Err()).NotTo(HaveOccurred())
			Expect(zRemRangeByRank.Val()).To(Equal(int64(2)))

			val, err := client.ZRangeWithScores("zset", 0, -1).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal([]redis.Z{{3, "three"}}))
		})

		It("should ZRemRangeByScore", func() {
			zAdd := client.ZAdd("zset", redis.Z{1, "one"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{2, "two"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{3, "three"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())

			zRemRangeByScore := client.ZRemRangeByScore("zset", "-inf", "(2")
			Expect(zRemRangeByScore.Err()).NotTo(HaveOccurred())
			Expect(zRemRangeByScore.Val()).To(Equal(int64(1)))

			val, err := client.ZRangeWithScores("zset", 0, -1).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal([]redis.Z{{2, "two"}, {3, "three"}}))
		})

		It("should ZRevRange", func() {
			zAdd := client.ZAdd("zset", redis.Z{1, "one"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{2, "two"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{3, "three"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())

			zRevRange := client.ZRevRange("zset", 0, -1)
			Expect(zRevRange.Err()).NotTo(HaveOccurred())
			Expect(zRevRange.Val()).To(Equal([]string{"three", "two", "one"}))

			zRevRange = client.ZRevRange("zset", 2, 3)
			Expect(zRevRange.Err()).NotTo(HaveOccurred())
			Expect(zRevRange.Val()).To(Equal([]string{"one"}))

			zRevRange = client.ZRevRange("zset", -2, -1)
			Expect(zRevRange.Err()).NotTo(HaveOccurred())
			Expect(zRevRange.Val()).To(Equal([]string{"two", "one"}))
		})

		It("should ZRevRangeWithScoresMap", func() {
			zAdd := client.ZAdd("zset", redis.Z{1, "one"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{2, "two"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{3, "three"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())

			val, err := client.ZRevRangeWithScores("zset", 0, -1).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal([]redis.Z{{3, "three"}, {2, "two"}, {1, "one"}}))

			val, err = client.ZRevRangeWithScores("zset", 2, 3).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal([]redis.Z{{1, "one"}}))

			val, err = client.ZRevRangeWithScores("zset", -2, -1).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal([]redis.Z{{2, "two"}, {1, "one"}}))
		})

		It("should ZRevRangeByScore", func() {
			zadd := client.ZAdd("zset", redis.Z{1, "one"})
			Expect(zadd.Err()).NotTo(HaveOccurred())
			zadd = client.ZAdd("zset", redis.Z{2, "two"})
			Expect(zadd.Err()).NotTo(HaveOccurred())
			zadd = client.ZAdd("zset", redis.Z{3, "three"})
			Expect(zadd.Err()).NotTo(HaveOccurred())

			vals, err := client.ZRevRangeByScore(
				"zset", redis.ZRangeByScore{Max: "+inf", Min: "-inf"}).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(vals).To(Equal([]string{"three", "two", "one"}))

			vals, err = client.ZRevRangeByScore(
				"zset", redis.ZRangeByScore{Max: "2", Min: "(1"}).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(vals).To(Equal([]string{"two"}))

			vals, err = client.ZRevRangeByScore(
				"zset", redis.ZRangeByScore{Max: "(2", Min: "(1"}).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(vals).To(Equal([]string{}))
		})

		It("should ZRevRangeByScoreWithScores", func() {
			zadd := client.ZAdd("zset", redis.Z{1, "one"})
			Expect(zadd.Err()).NotTo(HaveOccurred())
			zadd = client.ZAdd("zset", redis.Z{2, "two"})
			Expect(zadd.Err()).NotTo(HaveOccurred())
			zadd = client.ZAdd("zset", redis.Z{3, "three"})
			Expect(zadd.Err()).NotTo(HaveOccurred())

			vals, err := client.ZRevRangeByScoreWithScores(
				"zset", redis.ZRangeByScore{Max: "+inf", Min: "-inf"}).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(vals).To(Equal([]redis.Z{{3, "three"}, {2, "two"}, {1, "one"}}))
		})

		It("should ZRevRangeByScoreWithScoresMap", func() {
			zAdd := client.ZAdd("zset", redis.Z{1, "one"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{2, "two"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{3, "three"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())

			val, err := client.ZRevRangeByScoreWithScores(
				"zset", redis.ZRangeByScore{Max: "+inf", Min: "-inf"}).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal([]redis.Z{{3, "three"}, {2, "two"}, {1, "one"}}))

			val, err = client.ZRevRangeByScoreWithScores(
				"zset", redis.ZRangeByScore{Max: "2", Min: "(1"}).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal([]redis.Z{{2, "two"}}))

			val, err = client.ZRevRangeByScoreWithScores(
				"zset", redis.ZRangeByScore{Max: "(2", Min: "(1"}).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal([]redis.Z{}))
		})

		It("should ZRevRank", func() {
			zAdd := client.ZAdd("zset", redis.Z{1, "one"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{2, "two"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset", redis.Z{3, "three"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())

			zRevRank := client.ZRevRank("zset", "one")
			Expect(zRevRank.Err()).NotTo(HaveOccurred())
			Expect(zRevRank.Val()).To(Equal(int64(2)))

			zRevRank = client.ZRevRank("zset", "four")
			Expect(zRevRank.Err()).To(Equal(redis.Nil))
			Expect(zRevRank.Val()).To(Equal(int64(0)))
		})

		It("should ZScore", func() {
			zAdd := client.ZAdd("zset", redis.Z{1.001, "one"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())

			zScore := client.ZScore("zset", "one")
			Expect(zScore.Err()).NotTo(HaveOccurred())
			Expect(zScore.Val()).To(Equal(float64(1.001)))
		})

		It("should ZUnionStore", func() {
			zAdd := client.ZAdd("zset1", redis.Z{1, "one"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset1", redis.Z{2, "two"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())

			zAdd = client.ZAdd("zset2", redis.Z{1, "one"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset2", redis.Z{2, "two"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())
			zAdd = client.ZAdd("zset2", redis.Z{3, "three"})
			Expect(zAdd.Err()).NotTo(HaveOccurred())

			zUnionStore := client.ZUnionStore(
				"out", redis.ZStore{Weights: []int64{2, 3}}, "zset1", "zset2")
			Expect(zUnionStore.Err()).NotTo(HaveOccurred())
			Expect(zUnionStore.Val()).To(Equal(int64(3)))

			val, err := client.ZRangeWithScores("out", 0, -1).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal([]redis.Z{{5, "one"}, {9, "three"}, {10, "two"}}))
		})

	})

	//------------------------------------------------------------------------------

	Describe("watch/unwatch", func() {

		It("should WatchUnwatch", func() {
			var C, N = 10, 1000
			if testing.Short() {
				N = 100
			}

			err := client.Set("key", "0", 0).Err()
			Expect(err).NotTo(HaveOccurred())

			wg := &sync.WaitGroup{}
			for i := 0; i < C; i++ {
				wg.Add(1)

				go func() {
					defer GinkgoRecover()
					defer wg.Done()

					multi := client.Multi()
					defer multi.Close()

					for j := 0; j < N; j++ {
						val, err := multi.Watch("key").Result()
						Expect(err).NotTo(HaveOccurred())
						Expect(val).To(Equal("OK"))

						val, err = multi.Get("key").Result()
						Expect(err).NotTo(HaveOccurred())
						Expect(val).NotTo(Equal(redis.Nil))

						num, err := strconv.ParseInt(val, 10, 64)
						Expect(err).NotTo(HaveOccurred())

						cmds, err := multi.Exec(func() error {
							multi.Set("key", strconv.FormatInt(num+1, 10), 0)
							return nil
						})
						if err == redis.TxFailedErr {
							j--
							continue
						}
						Expect(err).NotTo(HaveOccurred())
						Expect(cmds).To(HaveLen(1))
						Expect(cmds[0].Err()).NotTo(HaveOccurred())
					}
				}()
			}
			wg.Wait()

			val, err := client.Get("key").Int64()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal(int64(C * N)))
		})

	})

	Describe("marshaling/unmarshaling", func() {

		type convTest struct {
			value  interface{}
			wanted string
			dest   interface{}
		}

		convTests := []convTest{
			{nil, "", nil},
			{"hello", "hello", new(string)},
			{[]byte("hello"), "hello", new([]byte)},
			{int(1), "1", new(int)},
			{int8(1), "1", new(int8)},
			{int16(1), "1", new(int16)},
			{int32(1), "1", new(int32)},
			{int64(1), "1", new(int64)},
			{uint(1), "1", new(uint)},
			{uint8(1), "1", new(uint8)},
			{uint16(1), "1", new(uint16)},
			{uint32(1), "1", new(uint32)},
			{uint64(1), "1", new(uint64)},
			{float32(1.0), "1", new(float32)},
			{float64(1.0), "1", new(float64)},
			{true, "1", new(bool)},
			{false, "0", new(bool)},
		}

		It("should convert to string", func() {
			for _, test := range convTests {
				err := client.Set("key", test.value, 0).Err()
				Expect(err).NotTo(HaveOccurred())

				s, err := client.Get("key").Result()
				Expect(err).NotTo(HaveOccurred())
				Expect(s).To(Equal(test.wanted))

				if test.dest == nil {
					continue
				}

				err = client.Get("key").Scan(test.dest)
				Expect(err).NotTo(HaveOccurred())
				Expect(deref(test.dest)).To(Equal(test.value))
			}
		})

	})

	Describe("json marshaling/unmarshaling", func() {
		BeforeEach(func() {
			value := &numberStruct{Number: 42}
			err := client.Set("key", value, 0).Err()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should marshal custom values using json", func() {
			s, err := client.Get("key").Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(s).To(Equal(`{"Number":42}`))
		})

		It("should scan custom values using json", func() {
			value := &numberStruct{}
			err := client.Get("key").Scan(value)
			Expect(err).To(BeNil())
			Expect(value.Number).To(Equal(42))
		})

	})

})

type numberStruct struct {
	Number int
}

func (s *numberStruct) MarshalBinary() ([]byte, error) {
	return json.Marshal(s)
}

func (s *numberStruct) UnmarshalBinary(b []byte) error {
	return json.Unmarshal(b, s)
}

func deref(viface interface{}) interface{} {
	v := reflect.ValueOf(viface)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v.Interface()
}
