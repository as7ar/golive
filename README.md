# Usage
 * weflab: `ws://localhost:8080/api/weflab?key=(id)`
 * soop: `ws://localhost:8080/api/soop?bjid=(bjid)&chat=(true/false)`

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