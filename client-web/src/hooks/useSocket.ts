import { useEffect, useState } from "react";
import { Message, MessageType } from "../types/message";

type Props = {
  host?: string;
  room: string;
  username: string;
};
const useSocket = ({ host, room, username }: Props) => {
  const url = `${host ? host : "ws://localhost:8084"}?room=${room}`;
  const [socket, setSocket] = useState<WebSocket | null>(null);

  useEffect(() => {
    const socketRes = new WebSocket(url);

    socketRes.addEventListener("open", () => {
      const msgType: MessageType = "join";
      const msg: Message = {
        username,
        text: "has joined the session",
        type: msgType,
      };
      socketRes.send(JSON.stringify(msg));
    });

    setSocket(socketRes);
  }, [url]);
  return socket;
};

export default useSocket;
