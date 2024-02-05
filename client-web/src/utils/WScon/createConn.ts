// import * as http from "http";
// import { Server as SocketIOServer, Socket } from "socket.io";

// // Flags represents the command-line flags that are passed to pairpad's client.
// class Flags {
//   Server: string;
//   File: string;
//   Room: string;

//   constructor(server: string, file: string, room: string) {
//     this.Server = server;
//     this.File = file;
//     this.Room = room;
//   }
// }

// // parseFlags parses command-line flags.
// function parseFlags(): Flags {
//   const serverAddr: string = process.env.npm_config_server || "localhost:8084";
//   const room: string = process.env.npm_config_room || "";
//   const file: string = process.env.npm_config_file || "";

//   return new Flags(serverAddr, file, room);
// }

// // createConn creates a Socket.IO connection.
// async function createConn(flags: Flags): Promise<Socket> {
//   const io = new SocketIOServer(http.createServer());

//   const u = new URL(`http://${flags.Server}`);
//   u.searchParams.set("room", flags.Room || "default");

//   console.log(`Connecting to ${u.toString()}...`);

//   return new Promise((resolve, reject) => {
//     io.on("connection", (socket: Socket) => {
//       resolve(socket);

//       socket.on("error", (error: any) => {
//         reject(error);
//       });
//     });
//   });
// }
