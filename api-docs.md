# Fitness Studio API 接口文档

> 基于 Go + Gin + SQLite + JWT 实现的健身工作室后端 API。所有接口前缀 `/api`，返回统一 JSON 格式。
>
> 默认启动端口：`:8080`
> 默认账号：
> - 管理员：`admin / admin123`
> - 会员：`member1 / 123456`、`member2 / 123456`

---

## 通用说明

### 认证方式

除登录和注册接口外，所有接口需要在 Header 中携带 JWT Token：

```
Authorization: Bearer <token>
```

### 角色权限说明

| 角色 | 说明 |
|------|------|
| **admin** | 管理员，可访问所有接口（包括会员管理、排课、统计等） |
| **member** | 会员，仅能操作自己的预约/候补/个人资料等 |
| **登录用户** | admin 或 member 都可访问 |

### 统一响应格式

成功响应：

```json
{
  "message": "操作描述",
  "data": "...（各接口具体字段见下）"
}
```

错误响应：

```json
{
  "error": "错误描述",
  "...其他错误上下文字段"
}
```

### 常用状态枚举

**预约状态（booking.status）**：
- `pending`：已预约，未签到
- `checked`：已签到
- `canceled`：已取消

**候补状态（waitlist.status）**：
- `waiting`：排队中
- `promoted`：已递补成功
- `canceled`：已取消

**会员状态（membership_status）**：
- `active`：会员有效
- `expired`：已过期

---

## 1. 健康检查

### GET /health

健康检查，无需鉴权。

**响应示例**：

```json
{
  "status": "ok",
  "message": "Fitness Studio API is running"
}
```

---

## 2. 认证模块 /api/auth

### POST /api/auth/login

用户登录（管理员或会员通用）。

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名 |
| password | string | 是 | 密码 |

**响应（200）**：

```json
{
  "token": "eyJhbGciOi...",
  "user": {
    "id": 1,
    "username": "admin",
    "name": "管理员",
    "role": "admin"
  }
}
```

**错误**：`400` 参数错误 / `401` 用户名或密码错误

---

### POST /api/auth/register

会员注册（仅会员角色可通过此接口注册，注册即赠送 1 个月会员）。

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 3-50 字符 |
| password | string | 是 | 最少 6 位 |
| name | string | 是 | 真实姓名 |
| phone | string | 否 | 手机号 |
| email | string | 否 | 邮箱 |
| gender | string | 否 | 性别 |

**响应（201）**：

```json
{
  "message": "注册成功，赠送1个月会员",
  "token": "eyJhbGciOi...",
  "user": {
    "id": 3,
    "username": "newuser",
    "name": "新会员",
    "membership_expire_at": "2026-07-24T10:00:00Z"
  }
}
```

**错误**：`400` 参数错误 / 用户名已存在

---

### GET /api/auth/me

获取当前登录用户信息。

**权限**：登录用户

**响应（200）**：

```json
{
  "user": {
    "id": 2,
    "username": "member1",
    "name": "王小明",
    "phone": "13800000001",
    "email": "",
    "gender": "",
    "role": "member",
    "membership_status": "active",
    "membership_expire_at": "2026-12-31T00:00:00Z",
    "days_until_expire": 190
  }
}
```

---

## 3. 会员模块 /api/members

### GET /api/members

会员列表（分页）。

**权限**：admin

**查询参数**：

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| page | int | 1 | 页码 |
| page_size | int | 10 | 每页条数 |
| status | string | - | `active` 有效 / `expired` 过期 |
| keyword | string | - | 搜索姓名/用户名/手机号模糊匹配 |

**响应（200）**：

```json
{
  "total": 2,
  "page": 1,
  "page_size": 10,
  "members": [
    {
      "id": 2,
      "username": "member1",
      "name": "王小明",
      "phone": "13800000001",
      "email": "",
      "gender": "",
      "membership_status": "active",
      "membership_expire_at": "2026-12-31T00:00:00Z",
      "days_until_expire": 190,
      "created_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

---

### GET /api/members/expiring-soon

即将到期会员列表。

**权限**：admin

**查询参数**：

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| days | int | 7 | 查询未来 N 天内到期 |

**响应（200）**：

```json
{
  "count": 1,
  "members": [
    {
      "id": 2,
      "name": "王小明",
      "phone": "13800000001",
      "membership_expire_at": "2026-06-30T00:00:00Z",
      "days_until_expire": 6
    }
  ]
}
```

---

### GET /api/members/:id

查看单个会员详情。

**权限**：admin

**路径参数**：`id` — 会员 ID

**响应（200）**：同单条会员结构（参见会员列表）

---

### PUT /api/members/:id

更新会员信息。

**权限**：
- admin：可改任意会员（路径传 id）
- member：只能改自己（路径 id 被忽略，以 token 为准）

**请求体**：所有字段可选

```json
{
  "name": "王小明更新",
  "phone": "13900001234",
  "email": "xm@example.com",
  "gender": "男"
}
```

**响应（200）**：

```json
{
  "message": "更新成功",
  "member": {
    "id": 2,
    "name": "王小明更新",
    "phone": "13900001234",
    "email": "xm@example.com",
    "gender": "男"
  }
}
```

---

### DELETE /api/members/:id

删除会员。

**权限**：admin

**响应（200）**：`{ "message": "删除成功" }`

---

### POST /api/members/renew

会员给自己续费。

**权限**：member

### POST /api/members/:id/renew

管理员给指定会员续费。

**权限**：admin

**请求体（两个接口共用）**：

```json
{
  "months": 3
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| months | int | 是 | 续多少个月（>=1） |

**响应（200）**：

```json
{
  "message": "续费成功",
  "months": 3,
  "new_expire_at": "2026-09-24T10:00:00Z",
  "days_until_expire": 92
}
```

---

## 4. 教练模块 /api/coaches

### GET /api/coaches

教练列表。

**权限**：登录用户

**响应（200）**：

```json
{
  "coaches": [
    {
      "id": 1,
      "name": "张教练",
      "phone": "13900000001",
      "specialty": "瑜伽 / 普拉提"
    }
  ]
}
```

---

### POST /api/coaches

新增教练。

**权限**：admin

**请求体**：

```json
{
  "name": "李教练",
  "phone": "13900000002",
  "specialty": "力量训练"
}
```

**响应（201）**：`{ "message": "创建成功", "coach": { ... } }`

---

### PUT /api/coaches/:id

更新教练。

**权限**：admin

**请求体**：同 POST /api/coaches

**响应（200）**：`{ "message": "更新成功", "coach": { ... } }`

---

### DELETE /api/coaches/:id

删除教练。

**权限**：admin

**响应（200）**：`{ "message": "删除成功" }`

---

## 5. 课程模块 /api/courses

### GET /api/courses

课程列表。

**权限**：登录用户

**响应（200）**：

```json
{
  "courses": [
    {
      "id": 1,
      "name": "瑜伽入门",
      "description": "零基础瑜伽课",
      "duration": 60
    }
  ]
}
```

---

### POST /api/courses

新增课程。

**权限**：admin

**请求体**：

```json
{
  "name": "动感单车",
  "description": "高强度有氧",
  "duration": 45
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 课程名 |
| description | string | 否 | 描述 |
| duration | int | 是 | 分钟数（>=1） |

**响应（201）**：`{ "message": "创建成功", "course": { ... } }`

---

### PUT /api/courses/:id

更新课程。

**权限**：admin

**响应（200）**：`{ "message": "更新成功", "course": { ... } }`

---

### DELETE /api/courses/:id

删除课程。

**权限**：admin

**响应（200）**：`{ "message": "删除成功" }`

---

## 6. 排课模块 /api/schedules

### GET /api/schedules

排课列表。

**权限**：登录用户

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| date | string | 指定日期，格式 `YYYY-MM-DD` |
| course_id | string | 按课程过滤 |
| coach_id | string | 按教练过滤 |

**响应（200）**：

```json
{
  "schedules": [
    {
      "id": 1,
      "course_id": 1,
      "course_name": "瑜伽入门",
      "coach_id": 1,
      "coach_name": "张教练",
      "start_time": "2026-06-25T10:00:00Z",
      "end_time": "2026-06-25T11:00:00Z",
      "capacity": 10,
      "booked_count": 3,
      "available_spots": 7,
      "room": "1号教室"
    }
  ]
}
```

---

### GET /api/schedules/:id

排课详情。

**权限**：登录用户

**响应（200）**：

```json
{
  "id": 1,
  "course_id": 1,
  "course_name": "瑜伽入门",
  "course_desc": "零基础瑜伽课",
  "coach_id": 1,
  "coach_name": "张教练",
  "coach_specialty": "瑜伽 / 普拉提",
  "start_time": "2026-06-25T10:00:00Z",
  "end_time": "2026-06-25T11:00:00Z",
  "capacity": 10,
  "booked_count": 3,
  "available_spots": 7,
  "room": "1号教室"
}
```

---

### POST /api/schedules

新增排课。

**权限**：admin

**请求体**：

```json
{
  "course_id": 1,
  "coach_id": 1,
  "start_time": "2026-06-25T10:00:00Z",
  "end_time": "2026-06-25T11:00:00Z",
  "capacity": 10,
  "room": "1号教室"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| course_id | uint | 是 | 课程 ID |
| coach_id | uint | 是 | 教练 ID |
| start_time | time | 是 | 开始时间 |
| end_time | time | 是 | 结束时间（必须晚于开始时间） |
| capacity | int | 是 | 人数上限（>=1） |
| room | string | 否 | 教室 |

**响应（201）**：

```json
{
  "message": "创建成功",
  "schedule": {
    "id": 5,
    "course_id": 1,
    "coach_id": 1,
    "start_time": "2026-06-25T10:00:00Z",
    "end_time": "2026-06-25T11:00:00Z",
    "capacity": 10,
    "room": "1号教室"
  }
}
```

---

### PUT /api/schedules/:id

更新排课。

**权限**：admin

**请求体**：同 POST /api/schedules

**响应（200）**：`{ "message": "更新成功" }`

---

### DELETE /api/schedules/:id

删除排课。

**权限**：admin

**响应（200）**：`{ "message": "删除成功" }`

---

### GET /api/schedules/:id/bookings

查看某节课程的所有预约 + 候补队列。

**权限**：admin

**响应（200）**：

```json
{
  "schedule": {
    "id": 1,
    "course_name": "瑜伽入门",
    "coach_name": "张教练",
    "start_time": "2026-06-25T10:00:00Z",
    "end_time": "2026-06-25T11:00:00Z",
    "capacity": 10,
    "booked_count": 3,
    "waitlist_count": 2,
    "checked_count": 1,
    "room": "1号教室"
  },
  "members": [
    {
      "booking_id": 1,
      "member_id": 2,
      "member_name": "王小明",
      "phone": "13800000001",
      "status": "checked",
      "checked_at": "2026-06-25T10:02:00Z",
      "booked_at": "2026-06-20T09:00:00Z"
    }
  ],
  "waitlist": [
    {
      "waitlist_id": 1,
      "member_id": 4,
      "member_name": "李小红",
      "phone": "13800000002",
      "position": 1,
      "status": "waiting",
      "created_at": "2026-06-23T10:00:00Z"
    }
  ]
}
```

---

## 7. 预约模块 /api/bookings

### GET /api/bookings

预约列表（管理员视角，支持筛选）。

**权限**：admin

**查询参数**：

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| schedule_id | string | - | 按排课过滤 |
| member_id | string | - | 按会员过滤 |
| status | string | - | pending / checked / canceled |
| page | int | 1 | 页码 |
| page_size | int | 20 | 每页条数 |

**响应（200）**：

```json
{
  "total": 10,
  "page": 1,
  "page_size": 20,
  "bookings": [
    {
      "id": 1,
      "schedule_id": 1,
      "course_name": "瑜伽入门",
      "member_id": 2,
      "member_name": "王小明",
      "start_time": "2026-06-25T10:00:00Z",
      "status": "pending",
      "checked_at": null,
      "created_at": "2026-06-20T09:00:00Z"
    }
  ]
}
```

---

### GET /api/bookings/my

当前登录会员的预约列表。

**权限**：member

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| status | string | 可选，pending / checked / canceled |

**响应（200）**：

```json
{
  "count": 2,
  "bookings": [
    {
      "id": 1,
      "schedule_id": 1,
      "course_name": "瑜伽入门",
      "coach_name": "张教练",
      "start_time": "2026-06-25T10:00:00Z",
      "end_time": "2026-06-25T11:00:00Z",
      "room": "1号教室",
      "status": "pending",
      "checked_at": null,
      "created_at": "2026-06-20T09:00:00Z"
    }
  ]
}
```

---

### POST /api/bookings

预约课程（或课程已满时加入候补队列）。

**权限**：member

**请求体**：

```json
{
  "schedule_id": 1
}
```

**响应 — 预约成功（201）**：

```json
{
  "message": "预约成功",
  "booking": {
    "id": 5,
    "schedule_id": 1,
    "course_name": "瑜伽入门",
    "coach_name": "张教练",
    "start_time": "2026-06-25T10:00:00Z",
    "end_time": "2026-06-25T11:00:00Z",
    "room": "1号教室",
    "status": "pending",
    "created_at": "2026-06-24T10:00:00Z"
  }
}
```

**响应 — 课程已满，加入候补（200）**：

```json
{
  "message": "课程已满，已加入候补队列",
  "queue_type": "waitlist",
  "waitlist": {
    "id": 3,
    "schedule_id": 1,
    "course_name": "瑜伽入门",
    "coach_name": "张教练",
    "start_time": "2026-06-25T10:00:00Z",
    "position": 1,
    "total_waiting": 1,
    "status": "waiting",
    "created_at": "2026-06-24T10:00:00Z"
  },
  "note": "有人取消预约时，将按候补顺序自动递补"
}
```

**错误场景**：
- `400` 会员已过期 → 附带 `membership_expire_at`
- `400` 已预约该课程
- `400` 已在候补队列 → 附带当前 `position`
- `400` 课程已开始
- `404` 排课不存在 / 会员不存在

---

### PUT /api/bookings/:id/cancel

取消预约。取消后如果有候补队列，会自动递补第一名。

**权限**：
- member：只能取消自己的预约
- admin：可取消任意预约

**响应（200）**：

```json
{
  "message": "取消预约成功",
  "booking": {
    "id": 1,
    "course_name": "瑜伽入门",
    "start_time": "2026-06-25T10:00:00Z",
    "status": "canceled"
  },
  "promoted": {
    "message": "已自动递补候补会员",
    "member_id": 4,
    "member_name": "李小红",
    "new_booking_id": 10,
    "waitlist_id": 1,
    "was_position": 1
  },
  "remaining_waitlist": 1
}
```

> 若没有可递补的候补，则响应中不会有 `promoted` 和 `remaining_waitlist` 字段。

---

### PUT /api/bookings/:id/checkin

签到。

**权限**：admin

**规则**：
- 开课前 30 分钟内至课程结束前可签到
- 已取消的预约无法签到
- 重复签到会返回错误

**响应（200）**：

```json
{
  "message": "签到成功",
  "booking": {
    "id": 1,
    "member_name": "王小明",
    "course_name": "瑜伽入门",
    "start_time": "2026-06-25T10:00:00Z",
    "checked_at": "2026-06-25T10:02:00Z",
    "status": "checked"
  }
}
```

---

## 8. 候补模块 /api/waitlist

### GET /api/waitlist/my

当前登录会员的候补记录。

**权限**：member

**响应（200）**：

```json
{
  "count": 1,
  "waitlist": [
    {
      "id": 1,
      "schedule_id": 1,
      "course_name": "瑜伽入门",
      "coach_name": "张教练",
      "start_time": "2026-06-25T10:00:00Z",
      "position": 1,
      "status": "waiting",
      "created_at": "2026-06-24T10:00:00Z"
    }
  ]
}
```

---

### PUT /api/waitlist/:id/cancel

取消候补。取消后剩余候替补位会自动重排。

**权限**：
- member：只能取消自己的候补
- admin：可取消任意候补

**响应（200）**：

```json
{
  "message": "取消候补成功",
  "waitlist": {
    "id": 1,
    "schedule_id": 1,
    "status": "canceled"
  },
  "remaining_waitlist": 0
}
```

---

### GET /api/waitlist/schedule/:id

查看某排课的候补队列。

**权限**：admin

**响应（200）**：

```json
{
  "count": 2,
  "waitlist": [
    {
      "waitlist_id": 1,
      "member_id": 4,
      "member_name": "李小红",
      "phone": "13800000002",
      "position": 1,
      "status": "waiting",
      "created_at": "2026-06-23T10:00:00Z"
    },
    {
      "waitlist_id": 2,
      "member_id": 5,
      "member_name": "赵小刚",
      "phone": "13800000003",
      "position": 2,
      "status": "waiting",
      "created_at": "2026-06-24T08:00:00Z"
    }
  ]
}
```

---

## 9. 统计模块 /api/stats

> 所有统计接口均需 admin 权限。

### GET /api/stats/dashboard

仪表盘概览。

**响应（200）**：

```json
{
  "members": {
    "total": 20,
    "active": 18,
    "expiring_soon": 3
  },
  "today": {
    "schedules": 5,
    "bookings": 42,
    "checkins": 30
  },
  "this_week": {
    "bookings": 210,
    "checkins": 180
  },
  "top_courses": [
    { "course_name": "瑜伽入门", "book_count": 60 },
    { "course_name": "动感单车", "book_count": 45 }
  ]
}
```

---

### GET /api/stats/weekly-attendance

本周各课程出勤率统计（周一 ~ 周日）。

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| date | string | 参考日期 `YYYY-MM-DD`，默认今天 |

**响应（200）**：

```json
{
  "week_start": "2026-06-22T00:00:00Z",
  "week_end": "2026-06-28T23:59:59Z",
  "summary": {
    "total_slots": 200,
    "total_booked": 150,
    "total_checked": 120,
    "booking_rate": 75.0,
    "attendance_rate": 60.0
  },
  "course_attendance": [
    {
      "WeekStart": "2026-06-22T00:00:00Z",
      "CourseName": "瑜伽入门",
      "TotalSlots": 60,
      "BookedCount": 50,
      "CheckedCount": 42,
      "AttendanceRate": 70.0
    }
  ]
}
```

---

### GET /api/stats/member-activity

本周会员活跃度排行榜（Top 20）。活跃度 = 签到数×10 + 预约数×3。

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| date | string | 参考日期 `YYYY-MM-DD`，默认今天 |

**响应（200）**：

```json
{
  "week_start": "2026-06-22T00:00:00Z",
  "week_end": "2026-06-28T23:59:59Z",
  "summary": {
    "active_members": 18,
    "total_bookings": 210,
    "total_checkins": 180,
    "avg_bookings_per_member": 11.67
  },
  "member_activities": [
    {
      "MemberID": 2,
      "MemberName": "王小明",
      "BookCount": 8,
      "CheckCount": 7,
      "ActivityScore": 94
    }
  ]
}
```

---

## 接口权限速查表

| 接口 | 方法 | 权限 |
|------|------|------|
| /health | GET | 公开 |
| /api/auth/login | POST | 公开 |
| /api/auth/register | POST | 公开 |
| /api/auth/me | GET | 登录用户 |
| /api/members | GET | admin |
| /api/members/expiring-soon | GET | admin |
| /api/members/:id | GET | admin |
| /api/members/:id | PUT | admin / member(本人) |
| /api/members/:id | DELETE | admin |
| /api/members/renew | POST | member |
| /api/members/:id/renew | POST | admin |
| /api/coaches | GET | 登录用户 |
| /api/coaches | POST | admin |
| /api/coaches/:id | PUT | admin |
| /api/coaches/:id | DELETE | admin |
| /api/courses | GET | 登录用户 |
| /api/courses | POST | admin |
| /api/courses/:id | PUT | admin |
| /api/courses/:id | DELETE | admin |
| /api/schedules | GET | 登录用户 |
| /api/schedules/:id | GET | 登录用户 |
| /api/schedules | POST | admin |
| /api/schedules/:id | PUT | admin |
| /api/schedules/:id | DELETE | admin |
| /api/schedules/:id/bookings | GET | admin |
| /api/bookings | GET | admin |
| /api/bookings/my | GET | member |
| /api/bookings | POST | member |
| /api/bookings/:id/cancel | PUT | admin / member(本人) |
| /api/bookings/:id/checkin | PUT | admin |
| /api/waitlist/my | GET | member |
| /api/waitlist/:id/cancel | PUT | admin / member(本人) |
| /api/waitlist/schedule/:id | GET | admin |
| /api/stats/dashboard | GET | admin |
| /api/stats/weekly-attendance | GET | admin |
| /api/stats/member-activity | GET | admin |
