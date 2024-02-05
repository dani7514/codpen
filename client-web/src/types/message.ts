import { Doc } from "../utils/CRDT/woot";
import { Operation } from "./operation";

export interface Message {
  username?: string;
  text?: string;
  type?: MessageType;
  ID?: string;
  operation?: Operation;
  document?: Doc;
}

// MessageType represents the type of the message.
export type MessageType =
  | "docSync"
  | "docReq"
  | "SiteID"
  | "join"
  | "users"
  | "operation";

// Currently, pairpad supports 5 message types:
// - docSync (for syncing documents)
// - docReq (for requesting documents)
// - SiteID (for generating site IDs)
// - join (for joining messages)
// - users (for the list of active users)

// Operation represents a CRDT operation.
