import { Controlled as CodeMirror } from "react-codemirror2";
import 'codemirror/lib/codemirror.css';
import 'codemirror/theme/material.css';
import 'codemirror/mode/javascript/javascript';
import 'codemirror/mode/xml/xml';
import 'codemirror/mode/css/css';

import { useEffect, useRef, useState } from "react";
import { Message } from "../types/message";
import { Character, Doc } from "../utils/CRDT/woot";
import useSocket from "../hooks/useSocket";

import { ToastContainer, toast } from 'react-toastify';
import 'react-toastify/dist/ReactToastify.css';
import Loader from "./Loader";


type OpType = {
  type: 'insert' | 'delete',
  index: number,
  ch?: string
} 

interface TextEditorProps {
  username: string;
  room: string;
  language: string;
  display: string;
  value: string;
  setValue: React.Dispatch<React.SetStateAction<string>>;
}

export default function TextEditor({ username, room, language, display, value, setValue }: TextEditorProps) {

console.log(username, room, language, display, value, setValue)

const socket = useSocket({room: `${room}-${language}`, username});

const doc = useRef(new Doc());
const hasSynced = useRef(false);
const [users, setUsers] = useState<string []>([]); // [username, siteID, cursorPos, selectionStart, selectionEnd]

console.log("Users: ", users)

  
  useEffect(() => {
    if (socket){
      socket.addEventListener('message', (event) => {
        const msg: Message = JSON.parse(event.data);
        console.log(`Message received: ${msg.type}`);
        switch (msg.type) {
          case 'docSync':
            if (!hasSynced.current) {
              hasSynced.current = true;
              console.log(`DOCSYNC RECEIVED, updating local doc ${msg.ID}`);
              const syncedCharacters: Character[] = msg?.document?.Characters?.map((c: any) => new Character(c.ID, c.Visible, c.Value, c.CP, c.CN))!;
              syncedCharacters.forEach((c: Character, index) => {
                doc.current.LocalInsert(c, index + 1);
              });
              setValue(doc.current.Content());
            }
            break;
          case 'docReq':
            console.log(`DOCREQ RECEIVED, sending local document to ${msg.ID}`);
            const replyMsg: Message = {
              type: 'docSync',
              document: doc.current,
              ID: msg.ID
            }
            socket.send(JSON.stringify(replyMsg));
            break;
          case 'SiteID':
            const siteID = parseInt(msg.text!);
            console.log(`SiteID RECEIVED, updating local SiteID ${msg.ID}`);
            Doc.SiteID = siteID;
            break;
          case 'users':
            console.log(`USERS RECEIVED, updating local users ${msg.text}`);
            const users = msg.text!.split(',');
            setUsers(users);
            break;
          case 'join':
            if (language === 'xml') {
              const message = `${msg.username} has joined the room`;
              toast.success(message,
                {
                  position: "bottom-right",
                  autoClose: 5000,
                  hideProgressBar: false,
                  closeOnClick: true,
                  pauseOnHover: true,
                  draggable: true,
                  progress: undefined,
                  theme: "colored",
                  }
                );
              }
              break;
          default:
            console.log(`Operation message with type: ${msg.operation?.type}`);
            switch (msg.operation?.type) {
              case "insert":
                const insertIndex = msg.operation.position;
                const insertValue = msg.operation.value!;
                
                doc.current.Insert(insertIndex, insertValue);
                setValue(doc.current.Content());
                
                break;
              case "delete":
                const deleteIndex = msg.operation.position;
                doc.current.Delete(deleteIndex);
                setValue(doc.current.Content());
                break;
              default:
                console.log(`Operation message with type: ${msg.operation?.type}`);
                break;
            }
            break;
        }
      });
    }
  }, [socket]);
  
  const performOperation = (opType: OpType) => {
    const { type, index, ch } = opType;
    let msg: Message | null = null;

    switch (type) {
      case 'insert':
        doc.current.Insert(index + 1, ch!); // TODO: maybe + 1
        setValue(doc.current.Content());
        msg = {
          type: 'operation',
          operation: {
            type: 'insert',
            position: index+1,
            value: ch!
          }
        }
        break;
      case 'delete':
        doc.current.Delete(index);
        setValue(doc.current.Content());
        msg = {
          type: 'operation',
          operation: {
            type: 'delete',
            position: index
          }
        }
        break;
      default:
        console.log(`Operation type: ${type}`);
        break;
    }
    if (socket && msg) {
      socket.send(JSON.stringify(msg!));
    }
  }

  if (socket === null) {
    return <Loader />;
  }

  return (
    <>
      <ToastContainer />
      <div className="grow basis-0 top-1/2 flex flex-col p-2 bg-[]">
          <div className="flex justify-between bg-[#3e3c44] text-gray-300 p-2 pl-4">
            {display}
          </div>
          <CodeMirror                                                                                                      
            value={value}
            options={{
              mode: language,
              theme: 'material',
              lineNumbers: true,
              electricChars: [],
              extraKeys: {
                  Enter: () => { } // Disable Enter key action
                }
            }}
            onKeyDown={(editor, event) => {
              if (event.key === 'Enter') {
                  // Prevent the default Enter key behavior (auto-indentation)
                  event.preventDefault();
                
                  // Get the cursor position
                  const cursor = editor.getCursor();
                  const index = editor.indexFromPos(cursor);

                  setValue(prevValue => `${prevValue.slice(0, index)}\n${prevValue.slice(index)}`);
                  
                  performOperation({ type: "insert", index, ch: '\n'});
              }
            }}
            onBeforeChange= {(editor, data, value) => {
              const changeType = data.origin;
              const cursorPosition = editor.getCursor();
              const index = editor.indexFromPos(cursorPosition);

              console.log(`Change type: ${changeType}, index: ${index}`);
              
              if  (changeType === '+delete') {
                performOperation({ type: "delete", index });
              }
              else if (changeType === '+input') {
                const ch = value.charAt(index);
                performOperation({ type: "insert", index, ch });
              }
            }}
          />
      </div>
    </>
  );
}