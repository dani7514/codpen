import { useSearchParams } from "react-router-dom"
import TextEditor from "./components/TextEditor";
import { useState } from "react";

const Edit = () => {
    const [searchParams] = useSearchParams();
    const username = searchParams.get('username') || 'default'; // TODO gen random name
    const room = searchParams.get('room') || 'default';


    const [html, setHtml] = useState('');
    const [css, setCss] = useState('');
    const [js, setJs] = useState('');

    const doc = `
    <html>
      <body>${html}</body>
      <style>${css}</style>
      <script>${js}</script>
    </html>`

  return (
    <>
      <div className="bg-[#86a789] font-mono h-[50vh] flex">
          <TextEditor username={username} room={room} language="xml" display="HTML" value={html} setValue={setHtml} />
          <TextEditor username={username} room={room} language="css" display="CSS" value={css} setValue={setCss} />
          <TextEditor username={username} room={room} language="javascript" display="JS" value={js} setValue={setJs}/>
      </div>
      <div className="h-[50vh]">
        <iframe 
          title="output"
          sandbox="allow-scripts"
          frameBorder="0"
          width='100%'
          height='100%'
          srcDoc={doc}
        />

      </div>
    </>
  )
}

export default Edit