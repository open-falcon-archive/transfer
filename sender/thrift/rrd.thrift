// filename: rrd.thrift
// namespace java com.xiaomi.infra.rrd.frontend.thrift
namespace go rrd

// for send
struct GraphItem {
  1: string uuid,   // 全局唯一的id
  2: double value,  // 值，只有double一种类型
  3: i64 timestamp, // unix时间戳，秒为单位
  4: string dsType, // 数据类型，GAUGE或者COUNTER
  5: i32 step,      // 数据汇报周期
  6: i32 heartbeat, // 心跳周期，rrd中的概念，数据如果超过heartbeat时间内都没有上来，相应时间点的数据就是NULL了
  7: string min,    // 数据的最小值，比该值小则认为不合法，丢弃，如果该值设置为"U"，表示无限制
  8: string max,    // 数据的最大值，比该值大则认为不合法，丢弃，如果该值设置为"U"，表示无限制
}

// for query
struct RRDData {
  1: i64 timestamp, // 数据项时间, unix时间戳, 秒为单位
  2: double value,  // 数据项取值, 只支持double类型
}

// query
struct QueryRequest {
  1: i64 startTs,   // 数据起始时间, unix时间戳, 秒为单位
  2: i64 endTs,     // 数据结束时间, unix时间戳, 秒为单位
  3: string cf,     // 归档函数, AVG|LAST|MIN|MAX
  4: string uuid,   // 数据项唯一标识
  5: string dsType, // 数据类型, GAUGE或者COUNTER
  6: i32 step,      // 数据周期, sec
}

struct QueryResponse {
  1: i64 startTs,   // 数据起始时间, unix.timestamp.in.sec
  2: i64 endTs,     // 数据结束时间, unix.timestamp.in.sec
  3: string cf,     // 归档函数, AVG|LAST|MIN|MAX
  4: string uuid,   // 数据项唯一标识
  5: string dsType, // 数据类型, GAUGE或者COUNTER
  6: i32 step,      // 数据周期, sec
  7: list<RRDData> values,  // 数据在指定时间段内的取值
}

// last
struct LastRequest {
  1: string uuid,   // 数据项唯一标识
  2: string dsType, // 数据类型, GAUGE或者COUNTER
  3: i32 step,      // 数据周期, sec
}

struct LastResponse {
  1: string uuid,   // 数据项唯一标识
  2: string dsType, // 数据类型, GAUGE或者COUNTER
  3: i32 step,      // 数据周期, sec
  4: RRDData value, // 最后收到的数据取值
}

service RRDHBaseBackend {
  void ping(),
  // 此处的返回值就是一个ErrorMessage即可，正常情况下留空，异常了就填充为错误消息
  string send(1:list<GraphItem> items)
  // 批量查询接口. 查询指定时间段内的数据、支持指定归档函数
  list<QueryResponse> query(1:list<QueryRequest> requests),
  // 最新数据查询接口. 查询指定数据项 最新上报的数据, COUNTER类型取差值、GAUGE类型取原值, 支持批量查询
  list<LastResponse> last(1:list<LastRequest> requests)
}