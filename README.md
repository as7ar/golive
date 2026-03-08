# Usage
 * weflab: `ws://localhost:8080/api/weflab?key=(id)`
 * soop: `ws://localhost:8080/api/soop?bjid=(bjid)&chat=(true/false)`
 * chzzk: `ws://localhost:8080/api/chzzk?id=(id)`

# Respond

## Weflab
```json
{
  "uid": "string",
  "uname": "string",
  "message": "string",
  "value": 0,
  "platform": "string",
  "type": "string"
}
```

## Soop
```json
{
  "Platform":"string",
  "name":"string",
  "Value": 0,
  "Message":"string",
  "Type":"string"
}
```

## Chzzk
```json
{
  "user": {
    "nickname": string,
    "user_id": string
  },
  "msg": string,
  "msgType":int,
  "msgStatus":int,
  "msgTime":long
}
```
