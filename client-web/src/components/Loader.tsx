import FadeLoader from "react-spinners/FadeLoader";

const override = {
  display: "block",
  margin: "0 auto",
  borderColor: "#0D5549",
};

function Loader() {

  return (
    <div className="sweet-loading flex justify-center items-center w-full h-full" style={{ position: "relative", }}>
    
      <FadeLoader
        color= "hsl(125, 16%, 59%)"
        loading={true}
        cssOverride={override}
        radius={75}
        aria-label="Loading Spinner"
        data-testid="loader"
      />
    </div>
  );
}

export default Loader;
