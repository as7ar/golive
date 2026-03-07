# Chzzk (Naver) Chat Reverse Notes

## 1. WebSocket 구조

### 인증 / 알림 WebSocket (socket.io)

```
wss://ssio10.nchat.naver.com/socket.io/?auth=...&EIO=3&transport=websocket
```

용도

* 로그인 세션 유지
* 개인 이벤트
* 알림 / 포인트

Heartbeat

```
client -> "2"   # ping
server -> "3"   # pong
```

이벤트 예시

```
42["personalMessage","{json...}"]
```

```
42 = socket.io event
event = personalMessage
payload = json string
```

채팅용 아님.

---

## 2. 실제 채팅 WebSocket

```
wss://kr-ss1.chat.naver.com/chat
```

socket.io 아님
별도 프로토콜 사용

---

## 3. 채팅 연결 과정

### 1. 인증

client

```json
{
 "ver":"3",
 "cmd":100,
 "svcid":"game",
 "cid":"CHANNEL_ID",
 "sid":null,
 "bdy":{
   "uid":"USER_HASH",
   "accTkn":"ACCESS_TOKEN",
   "auth":"SEND"
 },
 "tid":1
}
```

server

```
cmd:10100
sid: SESSION_ID
```

---

### 2. 최근 채팅 요청

client

```
cmd:5101
recentMessageCount:50
```

server

```
messageList:[...]
```

최근 채팅 반환

---

### 3. 실시간 채팅

서버가 push

```
messageTypeCode:1
```

일반 채팅 메시지

---

## 4. 채팅 데이터 구조

핵심 필드

```
nickname
content
messageTime
```

nickname 위치

```
profile JSON 내부
```

예

```
profile:"{...nickname...}"
extras:"{...}"
```

JSON 안에 JSON 구조 → 두 번 파싱 필요

---

## 5. 인코딩 문제

깨짐 예

```
ë°í¬íí¬
```

원인

```
UTF-8 → ISO-8859-1 잘못 디코딩
```

해결

```
UTF-8 decoding
```

---

## 6. 기본 루프 구조

```
connect
 ↓
auth
 ↓
request history
 ↓
read loop
 ↓
heartbeat loop
 ↓
reconnect
```

필수 요소

```
heartbeat
reconnect
json parsing
rate limit
```

---

## 7. 전체 구조 요약

```
REST
 ├─ chatChannelId
 └─ accessToken

WebSocket (socket.io)
 └─ 인증 / 알림

WebSocket (chat)
 ├─ 채팅 인증
 ├─ 채팅 기록
 └─ 실시간 채팅
```
