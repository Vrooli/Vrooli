import { useEffect } from 'react';
import { withRouter } from 'react-router-dom';

function ScrollToTopFunction({ history }) {
  useEffect(() => {
    const unlisten = history.listen(() => {
      window.scrollTo(0, 0);
    });
    return () => {
      unlisten();
    }
  }, [history]);

  return (null);
}

const ScrollToTop = withRouter(ScrollToTopFunction);
export { ScrollToTop };